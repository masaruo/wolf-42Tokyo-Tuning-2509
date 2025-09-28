package server

import (
	"backend/internal/db"
	"backend/internal/handler"
	"backend/internal/middleware"
	"backend/internal/repository"
	"backend/internal/service"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/riandyrn/otelchi"
)

type Server struct {
	Router *chi.Mux
}

func NewServer() (*Server, *sqlx.DB, error) {
	dbConn, err := db.InitDBConnection()
	if err != nil {
		return nil, nil, err
	}

	store := repository.NewStore(dbConn)

	authService := service.NewAuthService(store)
	orderService := service.NewOrderService(store)
	productService := service.NewProductService(store)
	robotService := service.NewRobotService(store)

	authHandler := handler.NewAuthHandler(authService)
	productHandler := handler.NewProductHandler(productService)
	orderHandler := handler.NewOrderHandler(orderService)
	robotHandler := handler.NewRobotHandler(robotService)

	userAuthMW := middleware.UserAuthMiddleware(store.SessionRepo)

	robotAPIKey := os.Getenv("ROBOT_API_KEY")
	if robotAPIKey == "" {
		log.Println("Warning: ROBOT_API_KEY is not set. Using default key 'test-robot-key'")
		robotAPIKey = "test-robot-key"
	}
	robotAuthMW := middleware.RobotAuthMiddleware(robotAPIKey)

	r := chi.NewRouter()
	
	// Add performance optimization middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Set performance headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			
			// Add compression for JSON responses
			if req.Header.Get("Accept-Encoding") != "" {
				w.Header().Set("Vary", "Accept-Encoding")
			}
			
			// Set cache headers for static content
			if req.URL.Path == "/api/health" {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			}
			
			next.ServeHTTP(w, req)
		})
	})
	
	r.Use(otelchi.Middleware(
		"backend-api",
		otelchi.WithChiRoutes(r),
		otelchi.WithFilter(func(req *http.Request) bool {
			return req.URL.Path != "/api/health"
		}),
	))

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	s := &Server{
		Router: r,
	}

	s.setupRoutes(authHandler, productHandler, orderHandler, robotHandler, userAuthMW, robotAuthMW)

	return s, dbConn, nil
}

func (s *Server) setupRoutes(
	authHandler *handler.AuthHandler,
	productHandler *handler.ProductHandler,
	orderHandler *handler.OrderHandler,
	robotHandler *handler.RobotHandler,
	userAuthMW func(http.Handler) http.Handler,
	robotAuthMW func(http.Handler) http.Handler,
) {
	s.Router.Post("/api/login", authHandler.Login)

	s.Router.Route("/api/v1", func(r chi.Router) {
		r.Use(userAuthMW)
		r.Post("/product", productHandler.List)
		r.Post("/product/post", productHandler.CreateOrders)
		r.Post("/orders", orderHandler.List)
		r.Get("/image", productHandler.GetImage)
	})

	s.Router.Route("/api/robot", func(r chi.Router) {
		r.Use(robotAuthMW)
		r.Get("/delivery-plan", robotHandler.GetDeliveryPlan)
		r.Patch("/orders/status", robotHandler.UpdateOrderStatus)
	})
}

func (s *Server) Run() {
	appPort := os.Getenv("PORT")
	if appPort == "" {
		appPort = "8080"
	}

	// Create optimized HTTP server with performance settings
	server := &http.Server{
		Addr:         ":" + appPort,
		Handler:      s.Router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		// Set maximum header size
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	log.Printf("Starting optimized server on :%s", appPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
