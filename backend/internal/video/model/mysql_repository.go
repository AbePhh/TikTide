package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type videoRecord struct {
	ID                  int64     `gorm:"column:id;primaryKey"`
	UserID              int64     `gorm:"column:user_id"`
	ObjectKey           string    `gorm:"column:object_key"`
	SourceURL           string    `gorm:"column:source_url"`
	Title               string    `gorm:"column:title"`
	CoverURL            string    `gorm:"column:cover_url"`
	DurationMS          int32     `gorm:"column:duration_ms"`
	AllowComment        int8      `gorm:"column:allow_comment"`
	Visibility          int8      `gorm:"column:visibility"`
	TranscodeStatus     int8      `gorm:"column:transcode_status"`
	AuditStatus         int8      `gorm:"column:audit_status"`
	TranscodeFailReason string    `gorm:"column:transcode_fail_reason"`
	AuditRemark         string    `gorm:"column:audit_remark"`
	PlayCount           int64     `gorm:"column:play_count"`
	LikeCount           int64     `gorm:"column:like_count"`
	CommentCount        int64     `gorm:"column:comment_count"`
	FavoriteCount       int64     `gorm:"column:favorite_count"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (videoRecord) TableName() string { return "t_video" }

type videoResourceRecord struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	VideoID    int64     `gorm:"column:video_id"`
	Resolution string    `gorm:"column:resolution"`
	FileURL    string    `gorm:"column:file_url"`
	FileSize   int64     `gorm:"column:file_size"`
	Bitrate    int32     `gorm:"column:bitrate"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (videoResourceRecord) TableName() string { return "t_video_resource" }

type videoHashtagRecord struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	VideoID   int64 `gorm:"column:video_id"`
	HashtagID int64 `gorm:"column:hashtag_id"`
}

func (videoHashtagRecord) TableName() string { return "t_video_hashtag" }

type hashtagRecord struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	UseCount  int64     `gorm:"column:use_count"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (hashtagRecord) TableName() string { return "t_hashtag" }

type draftRecord struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64     `gorm:"column:user_id"`
	ObjectKey    string    `gorm:"column:object_key"`
	CoverURL     string    `gorm:"column:cover_url"`
	Title        string    `gorm:"column:title"`
	TagNames     string    `gorm:"column:tag_names"`
	AllowComment int8      `gorm:"column:allow_comment"`
	Visibility   int8      `gorm:"column:visibility"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (draftRecord) TableName() string { return "t_draft" }

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) CreateVideo(ctx context.Context, video *Video, hashtagIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := toVideoRecord(video)
		if err := tx.Table("t_video").Create(&record).Error; err != nil {
			return fmt.Errorf("create video: %w", err)
		}

		if err := tx.Table("t_user_stats").
			Where("id = ?", video.UserID).
			Update("work_count", gorm.Expr("work_count + ?", 1)).Error; err != nil {
			return fmt.Errorf("increase work count: %w", err)
		}

		if len(hashtagIDs) == 0 {
			return nil
		}

		links := make([]videoHashtagRecord, 0, len(hashtagIDs))
		for _, hashtagID := range hashtagIDs {
			links = append(links, videoHashtagRecord{
				VideoID:   video.ID,
				HashtagID: hashtagID,
			})
		}

		if err := tx.Table("t_video_hashtag").Create(&links).Error; err != nil {
			return fmt.Errorf("create video hashtag links: %w", err)
		}

		if err := tx.Table("t_hashtag").
			Where("id IN ?", hashtagIDs).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Update("use_count", gorm.Expr("use_count + ?", 1)).Error; err != nil {
			return fmt.Errorf("increment hashtag use count: %w", err)
		}

		return nil
	})
}

func (r *MySQLRepository) GetVideoByID(ctx context.Context, videoID int64) (*Video, error) {
	var record videoRecord
	err := r.db.WithContext(ctx).
		Table("t_video").
		Where("id = ? AND deleted_at IS NULL", videoID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoNotFound
		}
		return nil, fmt.Errorf("query video: %w", err)
	}
	return fromVideoRecord(record), nil
}

func (r *MySQLRepository) ListVideosByIDs(ctx context.Context, videoIDs []int64) ([]Video, error) {
	if len(videoIDs) == 0 {
		return []Video{}, nil
	}

	var records []videoRecord
	if err := r.db.WithContext(ctx).
		Table("t_video").
		Where("id IN ? AND deleted_at IS NULL", videoIDs).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list videos by ids: %w", err)
	}

	index := make(map[int64]int, len(videoIDs))
	for i, videoID := range videoIDs {
		index[videoID] = i
	}

	items := make([]Video, 0, len(records))
	for _, record := range records {
		items = append(items, *fromVideoRecord(record))
	}
	sort.Slice(items, func(i, j int) bool {
		return index[items[i].ID] < index[items[j].ID]
	})
	return items, nil
}

func (r *MySQLRepository) ListVideosByUser(ctx context.Context, userID int64, cursor *time.Time, limit int, includeInvisible bool) ([]Video, error) {
	if userID <= 0 {
		return []Video{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := r.db.WithContext(ctx).
		Table("t_video").
		Where("user_id = ? AND deleted_at IS NULL", userID)

	if !includeInvisible {
		query = query.Where(
			"visibility = ? AND audit_status = ? AND transcode_status = ?",
			VisibilityPublic,
			AuditPassed,
			TranscodeSuccess,
		)
	}

	if cursor != nil {
		query = query.Where("created_at < ?", *cursor)
	}

	var records []videoRecord
	if err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list videos by user: %w", err)
	}

	items := make([]Video, 0, len(records))
	for _, record := range records {
		items = append(items, *fromVideoRecord(record))
	}
	return items, nil
}

func (r *MySQLRepository) ListRecommendVideos(ctx context.Context, limit int) ([]Video, error) {
	if limit <= 0 {
		limit = 500
	}
	if limit > 5000 {
		limit = 5000
	}

	var records []videoRecord
	if err := r.db.WithContext(ctx).
		Table("t_video").
		Where(
			"deleted_at IS NULL AND visibility = ? AND audit_status = ? AND transcode_status = ?",
			VisibilityPublic,
			AuditPassed,
			TranscodeSuccess,
		).
		Order("created_at DESC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list recommend videos: %w", err)
	}

	items := make([]Video, 0, len(records))
	for _, record := range records {
		items = append(items, *fromVideoRecord(record))
	}
	return items, nil
}

func (r *MySQLRepository) ListVideosForSearch(ctx context.Context, limit, offset int) ([]Video, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var records []videoRecord
	if err := r.db.WithContext(ctx).
		Table("t_video").
		Where("deleted_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list videos for search: %w", err)
	}

	items := make([]Video, 0, len(records))
	for _, record := range records {
		items = append(items, *fromVideoRecord(record))
	}
	return items, nil
}

func (r *MySQLRepository) ListHashtagNamesByVideoIDs(ctx context.Context, videoIDs []int64) (map[int64][]string, error) {
	result := make(map[int64][]string)
	if len(videoIDs) == 0 {
		return result, nil
	}

	type row struct {
		VideoID int64  `gorm:"column:video_id"`
		Name    string `gorm:"column:name"`
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Table("t_video_hashtag AS vh").
		Select("vh.video_id, h.name").
		Joins("JOIN t_hashtag AS h ON h.id = vh.hashtag_id").
		Where("vh.video_id IN ?", videoIDs).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list hashtag names by video ids: %w", err)
	}

	for _, item := range rows {
		result[item.VideoID] = append(result[item.VideoID], item.Name)
	}
	return result, nil
}

func (r *MySQLRepository) ListVideoResources(ctx context.Context, videoID int64) ([]VideoResource, error) {
	var records []videoResourceRecord
	if err := r.db.WithContext(ctx).
		Table("t_video_resource").
		Where("video_id = ?", videoID).
		Order("created_at ASC, id ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list video resources: %w", err)
	}

	items := make([]VideoResource, 0, len(records))
	for _, record := range records {
		items = append(items, VideoResource{
			ID:         record.ID,
			VideoID:    record.VideoID,
			Resolution: record.Resolution,
			FileURL:    record.FileURL,
			FileSize:   record.FileSize,
			Bitrate:    record.Bitrate,
			CreatedAt:  record.CreatedAt,
		})
	}
	return items, nil
}

func (r *MySQLRepository) IncreasePlayCount(ctx context.Context, videoID int64) error {
	result := r.db.WithContext(ctx).
		Table("t_video").
		Where("id = ? AND deleted_at IS NULL", videoID).
		Update("play_count", gorm.Expr("play_count + ?", 1))
	if result.Error != nil {
		return fmt.Errorf("increase play count: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrVideoNotFound
	}
	return nil
}

func (r *MySQLRepository) MarkVideoTranscoding(ctx context.Context, videoID int64) error {
	result := r.db.WithContext(ctx).
		Table("t_video").
		Where("id = ? AND deleted_at IS NULL AND transcode_status = ?", videoID, TranscodePending).
		Updates(map[string]any{
			"transcode_status":      TranscodeProcessing,
			"transcode_fail_reason": "",
		})
	if result.Error != nil {
		return fmt.Errorf("mark video transcoding: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInvalidTranscodeState
	}
	return nil
}

func (r *MySQLRepository) MarkVideoTranscodeSuccess(ctx context.Context, videoID int64, coverURL string, durationMS int32, resources []VideoResource) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Table("t_video").
			Where("id = ? AND deleted_at IS NULL AND transcode_status = ?", videoID, TranscodeProcessing).
			Updates(map[string]any{
				"cover_url":             coverURL,
				"duration_ms":           durationMS,
				"transcode_status":      TranscodeSuccess,
				"transcode_fail_reason": "",
			})
		if result.Error != nil {
			return fmt.Errorf("update transcode success: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrInvalidTranscodeState
		}

		resourceRecords := make([]videoResourceRecord, 0, len(resources))
		for _, resource := range resources {
			resourceRecords = append(resourceRecords, videoResourceRecord{
				VideoID:    videoID,
				Resolution: resource.Resolution,
				FileURL:    resource.FileURL,
				FileSize:   resource.FileSize,
				Bitrate:    resource.Bitrate,
			})
		}

		if err := tx.Table("t_video_resource").Create(&resourceRecords).Error; err != nil {
			return fmt.Errorf("create video resources: %w", err)
		}
		return nil
	})
}

func (r *MySQLRepository) MarkVideoTranscodeFailed(ctx context.Context, videoID int64, failReason string) error {
	result := r.db.WithContext(ctx).
		Table("t_video").
		Where("id = ? AND deleted_at IS NULL AND transcode_status = ?", videoID, TranscodeProcessing).
		Updates(map[string]any{
			"transcode_status":      TranscodeFailed,
			"transcode_fail_reason": failReason,
		})
	if result.Error != nil {
		return fmt.Errorf("mark video transcode failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInvalidTranscodeState
	}
	return nil
}

func (r *MySQLRepository) CountHashtagsByIDs(ctx context.Context, hashtagIDs []int64) (int64, error) {
	if len(hashtagIDs) == 0 {
		return 0, nil
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Table("t_hashtag").
		Where("id IN ?", hashtagIDs).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count hashtags: %w", err)
	}
	return count, nil
}

func (r *MySQLRepository) GetHashtagByID(ctx context.Context, hashtagID int64) (*Hashtag, error) {
	var record hashtagRecord
	err := r.db.WithContext(ctx).
		Table("t_hashtag").
		Where("id = ?", hashtagID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrHashtagNotFound
		}
		return nil, fmt.Errorf("query hashtag: %w", err)
	}

	return &Hashtag{
		ID:        record.ID,
		Name:      record.Name,
		UseCount:  record.UseCount,
		CreatedAt: record.CreatedAt,
	}, nil
}

func (r *MySQLRepository) CreateHashtag(ctx context.Context, name string) (*Hashtag, error) {
	record := hashtagRecord{Name: name}
	if err := r.db.WithContext(ctx).
		Table("t_hashtag").
		Where("name = ?", name).
		FirstOrCreate(&record).Error; err != nil {
		return nil, fmt.Errorf("create hashtag: %w", err)
	}

	return &Hashtag{
		ID:        record.ID,
		Name:      record.Name,
		UseCount:  record.UseCount,
		CreatedAt: record.CreatedAt,
	}, nil
}

func (r *MySQLRepository) ListHotHashtags(ctx context.Context, limit int) ([]Hashtag, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	var records []hashtagRecord
	if err := r.db.WithContext(ctx).
		Table("t_hashtag").
		Order("use_count DESC, id DESC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list hot hashtags: %w", err)
	}

	items := make([]Hashtag, 0, len(records))
	for _, record := range records {
		items = append(items, Hashtag{
			ID:        record.ID,
			Name:      record.Name,
			UseCount:  record.UseCount,
			CreatedAt: record.CreatedAt,
		})
	}
	return items, nil
}

func (r *MySQLRepository) ListHashtags(ctx context.Context, limit, offset int) ([]Hashtag, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var records []hashtagRecord
	if err := r.db.WithContext(ctx).
		Table("t_hashtag").
		Order("id ASC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list hashtags: %w", err)
	}

	items := make([]Hashtag, 0, len(records))
	for _, record := range records {
		items = append(items, Hashtag{
			ID:        record.ID,
			Name:      record.Name,
			UseCount:  record.UseCount,
			CreatedAt: record.CreatedAt,
		})
	}
	return items, nil
}

func (r *MySQLRepository) ListVideosByHashtag(ctx context.Context, hashtagID int64, cursor *time.Time, limit int) ([]HashtagVideo, error) {
	if limit <= 0 {
		limit = 20
	}

	query := r.db.WithContext(ctx).
		Table("t_video AS v").
		Select("v.id, v.user_id, v.title, v.object_key, v.source_url, v.cover_url, v.visibility, v.transcode_status, v.audit_status, v.created_at").
		Joins("JOIN t_video_hashtag AS vh ON vh.video_id = v.id").
		Where(
			"vh.hashtag_id = ? AND v.deleted_at IS NULL AND v.visibility = ? AND v.audit_status = ? AND v.transcode_status = ?",
			hashtagID,
			VisibilityPublic,
			AuditPassed,
			TranscodeSuccess,
		)

	if cursor != nil {
		query = query.Where("v.created_at < ?", *cursor)
	}

	var records []HashtagVideo
	if err := query.Order("v.created_at DESC").Limit(limit).Scan(&records).Error; err != nil {
		return nil, fmt.Errorf("list videos by hashtag: %w", err)
	}
	return records, nil
}

func (r *MySQLRepository) CreateDraft(ctx context.Context, draft *Draft) error {
	record := draftRecord{
		UserID:       draft.UserID,
		ObjectKey:    draft.ObjectKey,
		CoverURL:     draft.CoverURL,
		Title:        draft.Title,
		TagNames:     draft.TagNames,
		AllowComment: draft.AllowComment,
		Visibility:   draft.Visibility,
	}

	if err := r.db.WithContext(ctx).Table("t_draft").Create(&record).Error; err != nil {
		return fmt.Errorf("create draft: %w", err)
	}

	draft.ID = record.ID
	draft.CreatedAt = record.CreatedAt
	draft.UpdatedAt = record.UpdatedAt
	return nil
}

func (r *MySQLRepository) GetDraft(ctx context.Context, userID, draftID int64) (*Draft, error) {
	var record draftRecord
	if err := r.db.WithContext(ctx).
		Table("t_draft").
		Where("id = ? AND user_id = ?", draftID, userID).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDraftNotFound
		}
		return nil, fmt.Errorf("get draft: %w", err)
	}

	return &Draft{
		ID:           record.ID,
		UserID:       record.UserID,
		ObjectKey:    record.ObjectKey,
		CoverURL:     record.CoverURL,
		Title:        record.Title,
		TagNames:     record.TagNames,
		AllowComment: record.AllowComment,
		Visibility:   record.Visibility,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}, nil
}

func (r *MySQLRepository) UpdateDraft(ctx context.Context, draft *Draft) error {
	result := r.db.WithContext(ctx).
		Table("t_draft").
		Where("id = ? AND user_id = ?", draft.ID, draft.UserID).
		Updates(map[string]any{
			"object_key":    draft.ObjectKey,
			"cover_url":     draft.CoverURL,
			"title":         draft.Title,
			"tag_names":     draft.TagNames,
			"allow_comment": draft.AllowComment,
			"visibility":    draft.Visibility,
			"updated_at":    time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("update draft: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrDraftNotFound
	}

	updated, err := r.GetDraft(ctx, draft.UserID, draft.ID)
	if err != nil {
		return err
	}
	*draft = *updated
	return nil
}

func (r *MySQLRepository) ListDrafts(ctx context.Context, userID int64) ([]Draft, error) {
	var records []draftRecord
	if err := r.db.WithContext(ctx).
		Table("t_draft").
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list drafts: %w", err)
	}

	drafts := make([]Draft, 0, len(records))
	for _, record := range records {
		drafts = append(drafts, Draft{
			ID:           record.ID,
			UserID:       record.UserID,
			ObjectKey:    record.ObjectKey,
			CoverURL:     record.CoverURL,
			Title:        record.Title,
			TagNames:     record.TagNames,
			AllowComment: record.AllowComment,
			Visibility:   record.Visibility,
			CreatedAt:    record.CreatedAt,
			UpdatedAt:    record.UpdatedAt,
		})
	}
	return drafts, nil
}

func (r *MySQLRepository) DeleteDraft(ctx context.Context, userID, draftID int64) error {
	result := r.db.WithContext(ctx).
		Table("t_draft").
		Where("id = ? AND user_id = ?", draftID, userID).
		Delete(&draftRecord{})
	if result.Error != nil {
		return fmt.Errorf("delete draft: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrDraftNotFound
	}
	return nil
}

func toVideoRecord(video *Video) videoRecord {
	return videoRecord{
		ID:                  video.ID,
		UserID:              video.UserID,
		ObjectKey:           video.ObjectKey,
		SourceURL:           video.SourceURL,
		Title:               video.Title,
		CoverURL:            video.CoverURL,
		DurationMS:          video.DurationMS,
		AllowComment:        video.AllowComment,
		Visibility:          video.Visibility,
		TranscodeStatus:     video.TranscodeStatus,
		AuditStatus:         video.AuditStatus,
		TranscodeFailReason: video.TranscodeFailReason,
		AuditRemark:         video.AuditRemark,
		PlayCount:           video.PlayCount,
		LikeCount:           video.LikeCount,
		CommentCount:        video.CommentCount,
		FavoriteCount:       video.FavoriteCount,
	}
}

func fromVideoRecord(record videoRecord) *Video {
	return &Video{
		ID:                  record.ID,
		UserID:              record.UserID,
		ObjectKey:           record.ObjectKey,
		SourceURL:           record.SourceURL,
		Title:               record.Title,
		CoverURL:            record.CoverURL,
		DurationMS:          record.DurationMS,
		AllowComment:        record.AllowComment,
		Visibility:          record.Visibility,
		TranscodeStatus:     record.TranscodeStatus,
		AuditStatus:         record.AuditStatus,
		TranscodeFailReason: record.TranscodeFailReason,
		AuditRemark:         record.AuditRemark,
		PlayCount:           record.PlayCount,
		LikeCount:           record.LikeCount,
		CommentCount:        record.CommentCount,
		FavoriteCount:       record.FavoriteCount,
		CreatedAt:           record.CreatedAt,
		UpdatedAt:           record.UpdatedAt,
	}
}
