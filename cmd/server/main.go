package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/friend-diaries/backend/internal/application/auth"
	appgroup "github.com/friend-diaries/backend/internal/application/group"
	appnotification "github.com/friend-diaries/backend/internal/application/notification"
	appnote "github.com/friend-diaries/backend/internal/application/note"
	appreaction "github.com/friend-diaries/backend/internal/application/reaction"
	appreply "github.com/friend-diaries/backend/internal/application/reply"
	"github.com/friend-diaries/backend/internal/config"
	"github.com/friend-diaries/backend/internal/infrastructure/database"
	"github.com/friend-diaries/backend/internal/infrastructure/repository"
	httpinterface "github.com/friend-diaries/backend/internal/interfaces/http"
	"github.com/friend-diaries/backend/internal/interfaces/http/handler"
	"github.com/friend-diaries/backend/pkg/token"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("loading config", "error", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(cfg.Database.URL); err != nil {
		slog.Error("running migrations", "error", err)
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background(), cfg.Database.URL)
	if err != nil {
		slog.Error("connecting to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	tokenManager := token.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	userRepo := repository.NewUserRepository(pool)
	groupRepo := repository.NewGroupRepository(pool)
	noteRepo := repository.NewNoteRepository(pool)
	reactionRepo := repository.NewReactionRepository(pool)
	replyRepo := repository.NewReplyRepository(pool)
	notificationRepo := repository.NewNotificationRepository(pool)

	authService := auth.NewService(userRepo, tokenManager)
	groupService := appgroup.NewService(groupRepo, userRepo, notificationRepo)
	noteService := appnote.NewService(noteRepo, groupRepo, userRepo, notificationRepo)
	reactionService := appreaction.NewService(reactionRepo, noteRepo, groupRepo, notificationRepo)
	replyService := appreply.NewService(replyRepo, noteRepo, groupRepo, notificationRepo)
	notificationService := appnotification.NewService(notificationRepo)

	handlers := httpinterface.Handlers{
		Auth:         handler.NewAuthHandler(authService, tokenManager),
		User:         handler.NewUserHandler(userRepo),
		Group:        handler.NewGroupHandler(groupService),
		Note:         handler.NewNoteHandler(noteService),
		Reaction:     handler.NewReactionHandler(reactionService),
		Reply:        handler.NewReplyHandler(replyService),
		Notification: handler.NewNotificationHandler(notificationService),
	}

	router := httpinterface.NewRouter(handlers, tokenManager, cfg.CORS.AllowedOrigins)
	server := httpinterface.NewServer(cfg.Server.Port, router)

	if err := server.Run(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
