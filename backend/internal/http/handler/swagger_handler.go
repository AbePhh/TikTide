package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const swaggerHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>TikTide Swagger</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    html, body, #swagger-ui {
      margin: 0;
      padding: 0;
      width: 100%;
      height: 100%;
      background: #111827;
    }
    .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: '/swagger/openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        persistAuthorization: true,
        docExpansion: 'list',
        defaultModelsExpandDepth: 1,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: 'StandaloneLayout'
      });
    };
  </script>
</body>
</html>`

// SwaggerDoc 返回 OpenAPI 文档文件。
func SwaggerDoc(ctx *gin.Context) {
	content, err := os.ReadFile("docs/openapi.yaml")
	if err != nil {
		ctx.String(http.StatusInternalServerError, "swagger spec not found")
		return
	}

	ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", content)
}

// SwaggerHome 返回同源 Swagger UI 页面。
func SwaggerHome(ctx *gin.Context) {
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
}
