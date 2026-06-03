package http

import (
	"net/http"

	"github.com/friend-diaries/backend/internal/interfaces/http/handler"
	"github.com/friend-diaries/backend/internal/interfaces/http/middleware"
	"github.com/friend-diaries/backend/pkg/token"
	"github.com/go-chi/chi/v5"
	chimid "github.com/go-chi/chi/v5/middleware"
)

type Handlers struct {
	Auth         *handler.AuthHandler
	User         *handler.UserHandler
	Group        *handler.GroupHandler
	Note         *handler.NoteHandler
	Reaction     *handler.ReactionHandler
	Reply        *handler.ReplyHandler
	Notification *handler.NotificationHandler
	Friendship   *handler.FriendshipHandler
}

func NewRouter(h Handlers, tokenManager *token.Manager, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(chimid.Recoverer)
	r.Use(chimid.RequestID)
	r.Use(middleware.CORS(allowedOrigins))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", h.Auth.Register)
			r.Post("/login", h.Auth.Login)
			r.Post("/refresh", h.Auth.Refresh)
			r.Post("/logout", h.Auth.Logout)
		})

		r.Route("/share", func(r chi.Router) {
			r.Use(middleware.Auth(tokenManager))
			r.Get("/notes/{token}", h.Note.GetByShareToken)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(tokenManager))

			r.Get("/me", h.User.GetMe)
			r.Get("/users/profile/{appId}", h.User.GetPublicProfile)

			r.Route("/groups", func(r chi.Router) {
				r.Get("/", h.Group.List)
				r.Get("/invitations", h.Group.GetPendingInvitations)
				r.Post("/", h.Group.Create)

				r.Route("/{groupId}", func(r chi.Router) {
					r.Get("/", h.Group.Get)
					r.Get("/members", h.Group.GetMembers)
					r.Post("/invite", h.Group.Invite)
					r.Post("/accept", h.Group.AcceptInvitation)
					r.Post("/decline", h.Group.DeclineInvitation)
					r.Delete("/leave", h.Group.Leave)

					r.Route("/notes", func(r chi.Router) {
						r.Post("/", h.Note.Create)
						r.Get("/{date}", h.Note.ListByDate)
					})
				})
			})

			r.Route("/notes/{noteId}", func(r chi.Router) {
				r.Patch("/", h.Note.Update)
				r.Delete("/", h.Note.Delete)
				r.Post("/share", h.Note.CreateShareToken)

				r.Route("/reactions", func(r chi.Router) {
					r.Get("/", h.Reaction.List)
					r.Put("/", h.Reaction.Upsert)
					r.Delete("/", h.Reaction.Delete)
				})

				r.Route("/replies", func(r chi.Router) {
					r.Get("/", h.Reply.List)
					r.Post("/", h.Reply.Create)
				})
			})

			r.Route("/replies/{replyId}", func(r chi.Router) {
				r.Patch("/", h.Reply.Update)
				r.Delete("/", h.Reply.Delete)
			})

			r.Route("/notifications", func(r chi.Router) {
				r.Get("/", h.Notification.List)
				r.Post("/read-all", h.Notification.MarkAllRead)
				r.Post("/{notificationId}/read", h.Notification.MarkRead)
			})

			r.Route("/friends", func(r chi.Router) {
				r.Get("/", h.Friendship.ListFriends)
				r.Post("/request", h.Friendship.SendRequest)
				r.Get("/requests", h.Friendship.ListRequests)
				r.Post("/requests/{friendshipId}/accept", h.Friendship.Accept)
				r.Post("/requests/{friendshipId}/decline", h.Friendship.Decline)
				r.Delete("/{friendshipId}", h.Friendship.Remove)
				r.Put("/{friendshipId}/label", h.Friendship.SetLabel)
			})
		})
	})

	return r
}
