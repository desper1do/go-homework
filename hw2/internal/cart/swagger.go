package cart

import (
	"embed"
	"io/fs"
	"net/http"
)

// Спецификация и страница Swagger UI вшиваются в бинарник.
// Сам Swagger UI браузер загружает с CDN, поэтому для страницы нужен интернет.
//
//go:embed swagger
var swaggerFiles embed.FS

// swaggerHandler отдает содержимое папки swagger, по адресу /swagger/ открывается index.html.
func swaggerHandler() http.Handler {
	// Папка swagger есть в embed.FS, поэтому ошибки здесь быть не может.
	sub, err := fs.Sub(swaggerFiles, "swagger")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/swagger/", http.FileServerFS(sub))
}
