package handlers

import (
	"embed"
	"io/fs"

	"github.com/gofiber/fiber/v2"
)

//go:embed assets/similarity/*.svg
var staticAssets embed.FS

func SimilarityAsset(ctx *fiber.Ctx) error {
	fileName := ctx.Params("file")
	data, err := fs.ReadFile(staticAssets, "assets/similarity/"+fileName)
	if err != nil {
		return fiber.ErrNotFound
	}

	ctx.Set(fiber.HeaderContentType, "image/svg+xml")
	ctx.Set(fiber.HeaderCacheControl, "public, max-age=86400, immutable")
	return ctx.Send(data)
}
