package router

import (
	"ImageProcessor/internal/transport/handlers"
	"ImageProcessor/internal/transport/middleware"
	"github.com/wb-go/wbf/ginext"
	"go.uber.org/zap"
)

type Router struct {
	rout    *ginext.Engine
	handler *handlers.ImageHandler
	log     *zap.Logger
}

func NewRouter(mode string, handler *handlers.ImageHandler, log *zap.Logger) *Router {
	router := Router{
		rout:    ginext.New(mode),
		handler: handler,
		log:     log.Named("router"),
	}
	router.setupRouter()
	return &router
}

func (r *Router) setupRouter() {
	r.rout.Use(middleware.LoggingMiddleware(r.log))
	r.rout.POST("/upload", r.handler.UploadImage)
	r.rout.GET("/image/:id", r.handler.GetImage)
	r.rout.DELETE("/image/:id", r.handler.DeleteImage)

	r.rout.GET("/", func(c *ginext.Context) {
		c.File("./static/index.html")
	})

	r.rout.Static("/static", "static")
	r.rout.Static("/images", "./images")
}

func (r *Router) GetEngine() *ginext.Engine {
	return r.rout
}

func (r *Router) Start(addr string) error {
	return r.rout.Run(addr)
}
