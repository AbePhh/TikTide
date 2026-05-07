package model

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"

	"github.com/AbePhh/TikTide/backend/pkg/config"
)

type Repository interface {
	Initialize(ctx context.Context) error
	SearchUsers(ctx context.Context, req SearchRequest) (*SearchResult, error)
	SearchHashtags(ctx context.Context, req SearchRequest) (*SearchResult, error)
	SearchVideos(ctx context.Context, req SearchRequest) (*SearchResult, error)
	UpsertUserDocument(ctx context.Context, doc UserDocument) error
	UpsertHashtagDocument(ctx context.Context, doc HashtagDocument) error
	UpsertVideoDocument(ctx context.Context, doc VideoDocument) error
	DeleteUserDocument(ctx context.Context, documentID string) error
	DeleteHashtagDocument(ctx context.Context, documentID string) error
	DeleteVideoDocument(ctx context.Context, documentID string) error
}

type ElasticRepository struct {
	client        *elasticsearch.Client
	cfg           config.Config
	usersAlias    string
	hashtagsAlias string
	videosAlias   string
}

func NewElasticRepository(cfg config.Config) (*ElasticRepository, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.SearchAddresses,
		Username:  cfg.SearchUsername,
		Password:  cfg.SearchPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("new elasticsearch client: %w", err)
	}
	return &ElasticRepository{
		client:        client,
		cfg:           cfg,
		usersAlias:    strings.TrimSpace(cfg.SearchUsersAlias),
		hashtagsAlias: strings.TrimSpace(cfg.SearchHashtagsAlias),
		videosAlias:   strings.TrimSpace(cfg.SearchVideosAlias),
	}, nil
}

func (r *ElasticRepository) Initialize(ctx context.Context) error {
	if err := r.ensureIndex(ctx, r.usersAlias, r.buildUsersMapping()); err != nil {
		return err
	}
	if err := r.ensureIndex(ctx, r.hashtagsAlias, r.buildHashtagsMapping()); err != nil {
		return err
	}
	if err := r.ensureIndex(ctx, r.videosAlias, r.buildVideosMapping()); err != nil {
		return err
	}
	return nil
}

func (r *ElasticRepository) SearchUsers(ctx context.Context, req SearchRequest) (*SearchResult, error) {
	return r.search(ctx, r.usersAlias, req, []string{"username^4", "nickname^3", "signature"}, []string{"follower_count", "created_at", "id"})
}

func (r *ElasticRepository) SearchHashtags(ctx context.Context, req SearchRequest) (*SearchResult, error) {
	return r.search(ctx, r.hashtagsAlias, req, []string{"name^4"}, []string{"use_count", "created_at", "id"})
}

func (r *ElasticRepository) SearchVideos(ctx context.Context, req SearchRequest) (*SearchResult, error) {
	body := map[string]any{
		"size": sanitizeLimit(req.Limit, 20, 50),
		"query": map[string]any{
			"function_score": map[string]any{
				"query": map[string]any{
					"bool": map[string]any{
						"must": []any{
							map[string]any{
								"multi_match": map[string]any{
									"query":  strings.TrimSpace(req.Query),
									"fields": []string{"title^5", "author_username^3", "author_nickname^3", "hashtags^2"},
									"type":   "best_fields",
								},
							},
						},
						"filter": []any{
							map[string]any{"term": map[string]any{"visibility": 1}},
							map[string]any{"term": map[string]any{"audit_status": 1}},
							map[string]any{"term": map[string]any{"transcode_status": 2}},
						},
					},
				},
				"field_value_factor": map[string]any{
					"field":    "like_count",
					"modifier": "log1p",
					"missing":  0,
				},
				"boost_mode": "sum",
			},
		},
		"sort": []any{
			map[string]any{"_score": map[string]any{"order": "desc"}},
			map[string]any{"created_at": map[string]any{"order": "desc"}},
			map[string]any{"id": map[string]any{"order": "desc"}},
		},
	}
	return r.doSearch(ctx, r.videosAlias, req.Cursor, body)
}

func (r *ElasticRepository) UpsertUserDocument(ctx context.Context, doc UserDocument) error {
	return r.indexDocument(ctx, r.usersAlias, doc.ID, doc)
}

func (r *ElasticRepository) UpsertHashtagDocument(ctx context.Context, doc HashtagDocument) error {
	return r.indexDocument(ctx, r.hashtagsAlias, doc.ID, doc)
}

func (r *ElasticRepository) UpsertVideoDocument(ctx context.Context, doc VideoDocument) error {
	return r.indexDocument(ctx, r.videosAlias, doc.ID, doc)
}

func (r *ElasticRepository) DeleteUserDocument(ctx context.Context, documentID string) error {
	return r.deleteDocument(ctx, r.usersAlias, documentID)
}

func (r *ElasticRepository) DeleteHashtagDocument(ctx context.Context, documentID string) error {
	return r.deleteDocument(ctx, r.hashtagsAlias, documentID)
}

func (r *ElasticRepository) DeleteVideoDocument(ctx context.Context, documentID string) error {
	return r.deleteDocument(ctx, r.videosAlias, documentID)
}

func (r *ElasticRepository) search(ctx context.Context, index string, req SearchRequest, fields []string, sortFields []string) (*SearchResult, error) {
	sortItems := make([]any, 0, len(sortFields)+1)
	sortItems = append(sortItems, map[string]any{"_score": map[string]any{"order": "desc"}})
	for _, field := range sortFields {
		sortItems = append(sortItems, map[string]any{field: map[string]any{"order": "desc"}})
	}

	body := map[string]any{
		"size": sanitizeLimit(req.Limit, 20, 50),
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  strings.TrimSpace(req.Query),
				"fields": fields,
				"type":   "best_fields",
			},
		},
		"sort": sortItems,
	}
	return r.doSearch(ctx, index, req.Cursor, body)
}

func (r *ElasticRepository) doSearch(ctx context.Context, index, cursor string, body map[string]any) (*SearchResult, error) {
	if trimmed := strings.TrimSpace(cursor); trimmed != "" {
		values, err := decodeCursor(trimmed)
		if err != nil {
			return nil, err
		}
		body["search_after"] = values
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal search body: %w", err)
	}

	response, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(index),
		r.client.Search.WithBody(bytes.NewReader(payload)),
		r.client.Search.WithTrackTotalHits(false),
	)
	if err != nil {
		return nil, fmt.Errorf("search index %s: %w", index, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.IsError() {
		return nil, fmt.Errorf("search index %s status=%s body=%s", index, response.Status(), readBodySafe(response.Body))
	}

	type hit struct {
		ID   string `json:"_id"`
		Sort []any  `json:"sort"`
	}
	var result struct {
		Hits struct {
			Hits []hit `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}

	items := make([]SearchHit, 0, len(result.Hits.Hits))
	for _, item := range result.Hits.Hits {
		items = append(items, SearchHit{
			ID:         item.ID,
			SortValues: item.Sort,
		})
	}

	nextCursor := ""
	if len(items) > 0 {
		encoded, err := encodeCursor(items[len(items)-1].SortValues)
		if err != nil {
			return nil, err
		}
		nextCursor = encoded
	}

	return &SearchResult{
		Hits:       items,
		NextCursor: nextCursor,
	}, nil
}

func (r *ElasticRepository) indexDocument(ctx context.Context, index, documentID string, document any) error {
	payload, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}
	response, err := r.client.Index(
		index,
		bytes.NewReader(payload),
		r.client.Index.WithContext(ctx),
		r.client.Index.WithDocumentID(documentID),
		r.client.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("index document %s/%s: %w", index, documentID, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.IsError() {
		return fmt.Errorf("index document %s/%s status=%s body=%s", index, documentID, response.Status(), readBodySafe(response.Body))
	}
	return nil
}

func (r *ElasticRepository) deleteDocument(ctx context.Context, index, documentID string) error {
	response, err := r.client.Delete(
		index,
		documentID,
		r.client.Delete.WithContext(ctx),
		r.client.Delete.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("delete document %s/%s: %w", index, documentID, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusNotFound {
		return nil
	}
	if response.IsError() {
		return fmt.Errorf("delete document %s/%s status=%s body=%s", index, documentID, response.Status(), readBodySafe(response.Body))
	}
	return nil
}

func (r *ElasticRepository) ensureIndex(ctx context.Context, alias, mapping string) error {
	physical := alias + "_v1"

	existsResponse, err := r.client.Indices.Exists([]string{physical}, r.client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("check index %s: %w", physical, err)
	}
	defer func() { _ = existsResponse.Body.Close() }()

	if existsResponse.StatusCode == http.StatusNotFound {
		createResponse, createErr := r.client.Indices.Create(
			physical,
			r.client.Indices.Create.WithContext(ctx),
			r.client.Indices.Create.WithBody(strings.NewReader(mapping)),
		)
		if createErr != nil {
			return fmt.Errorf("create index %s: %w", physical, createErr)
		}
		defer func() { _ = createResponse.Body.Close() }()
		if createResponse.IsError() {
			return fmt.Errorf("create index %s status=%s body=%s", physical, createResponse.Status(), readBodySafe(createResponse.Body))
		}
	}

	aliasResponse, err := r.client.Indices.PutAlias(
		[]string{physical},
		alias,
		r.client.Indices.PutAlias.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("put alias %s => %s: %w", alias, physical, err)
	}
	defer func() { _ = aliasResponse.Body.Close() }()
	if aliasResponse.IsError() && aliasResponse.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("put alias %s => %s status=%s body=%s", alias, physical, aliasResponse.Status(), readBodySafe(aliasResponse.Body))
	}
	return nil
}

func (r *ElasticRepository) buildUsersMapping() string {
	return buildMapping(r.cfg.SearchUseIK, `{
  "properties": {
    "id": { "type": "keyword" },
    "username": { "type": "text", "analyzer": "%s", "fields": { "raw": { "type": "keyword" }, "suggest": { "type": "search_as_you_type" } } },
    "nickname": { "type": "text", "analyzer": "%s", "fields": { "raw": { "type": "keyword" }, "suggest": { "type": "search_as_you_type" } } },
    "signature": { "type": "text", "analyzer": "%s" },
    "avatar_url": { "type": "keyword", "index": false },
    "status": { "type": "byte" },
    "follower_count": { "type": "long" },
    "follow_count": { "type": "long" },
    "work_count": { "type": "long" },
    "created_at": { "type": "date" }
  }
}`)
}

func (r *ElasticRepository) buildHashtagsMapping() string {
	return buildMapping(r.cfg.SearchUseIK, `{
  "properties": {
    "id": { "type": "keyword" },
    "name": { "type": "text", "analyzer": "%s", "fields": { "raw": { "type": "keyword" }, "suggest": { "type": "search_as_you_type" } } },
    "use_count": { "type": "long" },
    "created_at": { "type": "date" }
  }
}`)
}

func (r *ElasticRepository) buildVideosMapping() string {
	return buildMapping(r.cfg.SearchUseIK, `{
  "properties": {
    "id": { "type": "keyword" },
    "title": { "type": "text", "analyzer": "%s", "fields": { "raw": { "type": "keyword" }, "suggest": { "type": "search_as_you_type" } } },
    "user_id": { "type": "keyword" },
    "author_username": { "type": "text", "analyzer": "%s", "fields": { "raw": { "type": "keyword" }, "suggest": { "type": "search_as_you_type" } } },
    "author_nickname": { "type": "text", "analyzer": "%s", "fields": { "raw": { "type": "keyword" }, "suggest": { "type": "search_as_you_type" } } },
    "hashtags": { "type": "text", "analyzer": "%s", "fields": { "raw": { "type": "keyword" }, "suggest": { "type": "search_as_you_type" } } },
    "cover_url": { "type": "keyword", "index": false },
    "play_count": { "type": "long" },
    "like_count": { "type": "long" },
    "comment_count": { "type": "long" },
    "favorite_count": { "type": "long" },
    "visibility": { "type": "byte" },
    "audit_status": { "type": "byte" },
    "transcode_status": { "type": "byte" },
    "created_at": { "type": "date" }
  }
}`)
}

func buildMapping(useIK bool, propertiesTemplate string) string {
	analyzer := "standard"
	if useIK {
		analyzer = "ik_smart"
	}
	properties := formatPropertiesTemplate(propertiesTemplate, analyzer)
	return `{
  "settings": {
    "analysis": {
      "analyzer": {
        "default_cn": { "type": "` + analyzer + `" }
      }
    }
  },
  "mappings": ` + properties + `
}`
}

func formatPropertiesTemplate(template, analyzer string) string {
	switch strings.Count(template, "%s") {
	case 0:
		return template
	case 1:
		return fmt.Sprintf(template, analyzer)
	case 2:
		return fmt.Sprintf(template, analyzer, analyzer)
	case 3:
		return fmt.Sprintf(template, analyzer, analyzer, analyzer)
	case 4:
		return fmt.Sprintf(template, analyzer, analyzer, analyzer, analyzer)
	default:
		args := make([]any, 0, strings.Count(template, "%s"))
		for i := 0; i < strings.Count(template, "%s"); i++ {
			args = append(args, analyzer)
		}
		return fmt.Sprintf(template, args...)
	}
}

func sanitizeLimit(value, fallback, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func encodeCursor(values []any) (string, error) {
	if len(values) == 0 {
		return "", nil
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal search cursor: %w", err)
	}
	return base64.StdEncoding.EncodeToString(payload), nil
}

func decodeCursor(cursor string) ([]any, error) {
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("decode search cursor: %w", err)
	}
	values := make([]any, 0)
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("unmarshal search cursor: %w", err)
	}
	for i := range values {
		switch typed := values[i].(type) {
		case float64:
			if typed == float64(int64(typed)) {
				values[i] = int64(typed)
			}
		case string:
			if parsed, parseErr := strconv.ParseInt(typed, 10, 64); parseErr == nil {
				values[i] = parsed
			}
		}
	}
	return values, nil
}

func readBodySafe(reader io.Reader) string {
	body, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}
	return string(body)
}
