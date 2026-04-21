package bootstrap

import (
	"log"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/auth"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/config"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/controller"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/middleware"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/platform/bus"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/platform/email"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/platform/ws"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/repo"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/route"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

type Repos struct {
	repo.UserRepo
	repo.EmailVerificationRepo
	repo.PasswordResetRepo
	repo.ProductRepo
	repo.CartRepo
	repo.ProvinceRepo
	repo.OrderRepo
	repo.CategoryRepo
}

type Services struct {
	service.AdminAuthService
	service.AuthService
	service.UserService
	service.AdminUserService
	service.MediaService
	service.ProductService
	service.CartService
	service.ProvinceService
	service.OrderService
	service.CategoryService
}
type Controllers struct {
	controller.AdminAuthController
	controller.AuthController
	controller.UserController
	controller.AdminUserController
	controller.MediaController
	controller.ProductController
	controller.CartController
	controller.ProvinceController
	controller.OrderController
	controller.CategoryController
}

func initRepos(client *mongo.Client, db *mongo.Database) *Repos {
	return &Repos{
		UserRepo:              repo.NewUserRepo(db),
		EmailVerificationRepo: repo.NewEmailVerificationRepo(db),
		PasswordResetRepo:     repo.NewPasswordResetRepo(db),
		ProductRepo:           repo.NewProductRepo(db),
		CartRepo:              repo.NewCartRepo(db),
		ProvinceRepo:          repo.NewProvinceRepo(db),
		OrderRepo:             repo.NewOrderRepo(db),
		CategoryRepo:          repo.NewCategoryRepo(db),
	}
}

func initServices(repos *Repos, redisClient *redis.Client, emailSender email.Sender, eventBus bus.EventBus, tokenService *auth.TokenService) *Services {
	services := &Services{
		AuthService:     service.NewAuthService(repos.UserRepo, repos.EmailVerificationRepo, repos.PasswordResetRepo, emailSender, redisClient, tokenService),
		UserService:     service.NewUserService(repos.UserRepo, eventBus, redisClient),
		MediaService:    service.NewMediaService(),
		ProductService:  service.NewProductService(repos.ProductRepo, repos.UserRepo, eventBus, redisClient),
		CartService:     service.NewCartService(repos.CartRepo, repos.ProductRepo, repos.UserRepo, eventBus, redisClient),
		ProvinceService: service.NewProvinceService(repos.ProvinceRepo),
		OrderService:    service.NewOrderService(repos.OrderRepo, repos.ProductRepo, repos.ProvinceRepo, repos.UserRepo),
		CategoryService: service.NewCategoryService(repos.CategoryRepo),
		//MembershipService:   service.NewMembershipService(repos.MembershipRepo, redisClient),
		//ReputationService:   service.NewReputationService(repos.UserRepo, eventBus),
		//NotificationService: service.NewNotificationService(repos.NotificationRepo, repos.UserRepo, repos.PostRepo, repos.CommentRepo, repos.CommunityRepo, eventBus, redisClient),
		//ChannelService:      service.NewChannelService(repos.ChannelRepo, eventBus),
		//MessageService:      service.NewMessageService(repos.MessageRepo, repos.ChannelRepo, repos.UserRepo, eventBus, redisClient),
		//PostHistoryService:  service.NewPostHistoryService(repos.PostHistoryRepo),
		//ReportService:       service.NewReportService(repos.ReportRepo),
		//CommentService:      service.NewCommentService(repos.CommentRepo, repos.UserRepo, repos.CommunityRepo, repos.PostRepo, eventBus),
		//CommunityService:    service.NewCommunityService(repos.CommunityRepo, repos.MembershipRepo, repos.PostRepo, repos.UserRepo, eventBus),
	}

	//// Set MembershipService in CommunityService to get real-time member count
	//if communitySvc, ok := services.CommunityService.(interface {
	//	SetMembershipService(service.MembershipService)
	//}); ok {
	//	communitySvc.SetMembershipService(services.MembershipService)
	//}
	//
	//// VoteService needs to be created first as PostService and CommentService depend on it
	//services.VoteService = service.NewVoteService(repos.VoteRepo, repos.PostRepo, repos.CommentRepo, eventBus)
	//
	//// PostService and CommentService need VoteService
	//services.PostService = service.NewPostService(repos.PostRepo, services.VoteService, repos.PollVoteRepo, repos.UserRepo, repos.CommunityRepo, repos.MembershipRepo, repos.SavedPostRepo, repos.ReportRepo, eventBus)
	//
	//// DraftService needs PostService
	//services.DraftService = service.NewDraftService(repos.DraftRepo, repos.PostRepo, services.PostService)
	//
	//// ModerationService needs CommunityRepo for checking PostRequireApproval
	//services.ModerationService = service.NewModerationService(repos.PostRepo, repos.CommentRepo, repos.UserRepo, repos.CommunityRepo, geminiClient, eventBus, &config.Cfg.Gemini)
	//
	//// AdminUserService for admin operations
	services.AdminUserService = service.NewAdminUserService(repos.UserRepo)
	//services.AdminCommunityService = service.NewAdminCommunityService(repos.CommunityRepo)
	//services.AdminStatsService = service.NewAdminStatsService(repos.UserRepo, repos.CommunityRepo, repos.PostRepo, repos.CommentRepo, repos.ReportRepo)
	services.AdminAuthService = service.NewAdminAuthService(repos.UserRepo, redisClient, tokenService)

	return services
}

func initControllers(services *Services, wsHub *ws.Hub, db *mongo.Database) *Controllers {
	return &Controllers{
		AuthController:     *controller.NewAuthController(services.AuthService),
		UserController:     *controller.NewUserController(services.UserService),
		MediaController:    *controller.NewMediaController(services.MediaService),
		ProductController:  *controller.NewProductController(services.ProductService),
		CartController:     *controller.NewCartController(services.CartService),
		ProvinceController: *controller.NewProvinceController(services.ProvinceService),
		OrderController:    *controller.NewOrderController(services.OrderService),
		CategoryController: *controller.NewCategoryController(services.CategoryService),
		//CommunityController:      *controller.NewCommunityController(services.CommunityService),
		//MembershipController:     *controller.NewMembershipController(services.MembershipService),
		//PostController:           *controller.NewPostController(services.PostService),
		//VoteController:           *controller.NewVoteController(services.VoteService), // Added VoteController
		//CommentController:        *controller.NewCommentController(services.CommentService),
		//NotificationController:   *controller.NewNotificationController(services.NotificationService),
		//WebSocketController:      *controller.NewWebSocketController(wsHub),
		//ChannelController:        *controller.NewChannelController(services.ChannelService),
		//MessageController:        *controller.NewMessageController(services.MessageService),
		//PostHistoryController:    *controller.NewPostHistoryController(services.PostHistoryService),
		//DraftController:          *controller.NewDraftController(services.DraftService),
		//ReportController:         *controller.NewReportController(services.ReportService),
		AdminUserController: *controller.NewAdminUserController(services.AdminUserService),
		//AdminCommunityController: *controller.NewAdminCommunityController(services.AdminCommunityService),
		//AdminStatsController:     *controller.NewAdminStatsController(services.AdminStatsService),
		AdminAuthController: *controller.NewAdminAuthController(services.AdminAuthService),
		//DebugController:          *controller.NewDebugController(db),
	}
}

func initRoutes(controllers *Controllers, r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// ⚠️ DEBUG ONLY - Remove in production
	//r.POST("/debug/create-admin", controllers.DebugController.CreateAdminUser)

	api := r.Group("/api")
	api.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Welcome to LKForum API!"})
	})

	route.RegisterAuthRoutes(api, &controllers.AuthController)
	route.RegisterUserRoutes(api, &controllers.UserController)
	route.RegisterMediaRoutes(api, &controllers.MediaController)
	route.RegisterCartRoutes(api, &controllers.CartController)
	route.RegisterProductRoutes(api, &controllers.ProductController)
	route.RegisterProvinceRoutes(api, &controllers.ProvinceController)
	route.RegisterOrderRoutes(api, &controllers.OrderController)
	route.RegisterCategoryRoutes(api, &controllers.CategoryController)
	//route.RegisterCommunityRoutes(api, &controllers.CommunityController)
	//route.RegisterMembershipRoutes(api, &controllers.MembershipController)
	//route.RegisterPostRoutes(api, &controllers.PostController)
	//route.RegisterVoteRoutes(api, &controllers.VoteController) // Added VoteRoutes
	//route.RegisterCommentRoutes(api, &controllers.CommentController)
	//route.RegisterNotificationRoutes(api, &controllers.NotificationController)
	//route.RegisterWebSocketRoutes(api, &controllers.WebSocketController)
	//route.RegisterChannelRoutes(api, &controllers.ChannelController)
	//route.RegisterMessageRoutes(api, &controllers.MessageController)
	//route.RegisterPostHistoryRoutes(api, &controllers.PostHistoryController)
	//route.RegisterDraftRoutes(api, &controllers.DraftController)
	//route.RegisterReportRoutes(api, &controllers.ReportController)
	route.RegisterAdminAuthRoutes(api, &controllers.AdminAuthController)
	route.RegisterAdminUserRoutes(api, &controllers.AdminUserController)
	//route.RegisterAdminCommunityRoutes(api, &controllers.CommunityController, &controllers.AdminCommunityController)
	//route.RegisterAdminReportRoutes(api, &controllers.ReportController)
	//route.RegisterAdminStatsRoutes(api, &controllers.AdminStatsController)
}

func Init() (*gin.Engine, error) {
	config.LoadConfig()
	auth.InitGoogleOAuthConfig()

	redisClient := config.NewRedisClient()

	tokenService, err := InitializeTokenService(redisClient)
	if err != nil {
		log.Printf("Warning: Token invalidation service not available: %v\n", err)
	}

	client := config.NewMongoClient()
	db := client.Database(config.Cfg.DBName)
	router := gin.Default()
	router.MaxMultipartMemory = 10 << 20 // 10 MB

	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := []string{"http://localhost:5173", "http://localhost:5174"}
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	eventBus := bus.NewEventBus()
	wsHub := ws.NewHub(eventBus)
	emailSender := email.NewSMTPSender()

	//// Initialize Gemini client for content moderation
	//geminiClient, err := gemini.NewGeminiClient(&config.Cfg.Gemini)
	//if err != nil {
	//	log.Printf("Warning: Gemini client initialization failed: %v. Content moderation will be disabled.", err)
	//}

	repos := initRepos(client, db)
	services := initServices(repos, redisClient, emailSender, eventBus, tokenService)
	controllers := initControllers(services, wsHub, db)

	// Inject userRepo into middleware for settings caching
	middleware.SetUserRepo(repos.UserRepo)

	initRoutes(controllers, router)

	// Start background services
	go wsHub.Start()
	//services.ReputationService.Start()
	//services.NotificationService.Start()
	//services.MessageService.Start()
	//services.ChannelService.Start()
	//services.CommunityService.Start()
	//services.ModerationService.Start() // Start content moderation service

	return router, nil
}
