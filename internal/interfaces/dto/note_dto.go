package dto

import (
	"time"

	"github.com/friend-diaries/backend/internal/domain/note"
	"github.com/friend-diaries/backend/internal/domain/reaction"
	"github.com/friend-diaries/backend/internal/domain/reply"
	"github.com/google/uuid"
)

type CreateNoteRequest struct {
	Content  string   `json:"content"`
	Date     string   `json:"date"`
	MediaIDs []string `json:"media_ids,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
}

func (r *CreateNoteRequest) Validate() error {
	if r.Content == "" && len(r.MediaIDs) == 0 {
		return errRequired("content")
	}
	if len(r.Content) > 10000 {
		return errMaxLength("content", 10000)
	}
	if r.ClientID != "" {
		if _, err := uuid.Parse(r.ClientID); err != nil {
			return errInvalid("client_id")
		}
	}
	return nil
}

type UpdateNoteRequest struct {
	Content string `json:"content"`
}

func (r *UpdateNoteRequest) Validate() error {
	if r.Content == "" {
		return errRequired("content")
	}
	return nil
}

type NoteMediaResponse struct {
	ID              uuid.UUID `json:"id"`
	MediaURL        string    `json:"media_url"`
	MimeType        string    `json:"mime_type"`
	MediaKind       string    `json:"media_kind"`
	DurationSeconds *float64  `json:"duration_seconds,omitempty"`
}

type NoteResponse struct {
	ID        uuid.UUID           `json:"id"`
	ClientID  *uuid.UUID          `json:"client_id,omitempty"`
	GroupID   uuid.UUID           `json:"group_id"`
	AuthorID  uuid.UUID           `json:"author_id"`
	DiaryDate string              `json:"diary_date"`
	Content   string              `json:"content"`
	IsEdited  bool                `json:"is_edited"`
	IsDeleted bool                `json:"is_deleted"`
	Media     []NoteMediaResponse `json:"media"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

func ToNoteResponse(n *note.Note) NoteResponse {
	content := n.Content
	if n.IsDeleted {
		content = "This note was deleted."
	}
	return NoteResponse{
		ID:        n.ID,
		ClientID:  n.ClientID,
		GroupID:   n.GroupID,
		AuthorID:  n.AuthorID,
		DiaryDate: n.DiaryDate.Format("2006-01-02"),
		Content:   content,
		IsEdited:  n.IsEdited,
		IsDeleted: n.IsDeleted,
		Media:     []NoteMediaResponse{},
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
}

type ReactionResponse struct {
	ID           uuid.UUID     `json:"id"`
	NoteID       uuid.UUID     `json:"note_id"`
	UserID       uuid.UUID     `json:"user_id"`
	ReactionType reaction.Type `json:"reaction_type"`
	CreatedAt    time.Time     `json:"created_at"`
}

func ToReactionResponse(r *reaction.Reaction) ReactionResponse {
	return ReactionResponse{
		ID:           r.ID,
		NoteID:       r.NoteID,
		UserID:       r.UserID,
		ReactionType: r.ReactionType,
		CreatedAt:    r.CreatedAt,
	}
}

type UpsertReactionRequest struct {
	ReactionType reaction.Type `json:"reaction_type"`
}

type CreateReplyRequest struct {
	Content string `json:"content"`
}

func (r *CreateReplyRequest) Validate() error {
	if r.Content == "" {
		return errRequired("content")
	}
	if len(r.Content) > 2000 {
		return errMaxLength("content", 2000)
	}
	return nil
}

type UpdateReplyRequest struct {
	Content string `json:"content"`
}

func (r *UpdateReplyRequest) Validate() error {
	return (&CreateReplyRequest{Content: r.Content}).Validate()
}

type ReplyResponse struct {
	ID        uuid.UUID `json:"id"`
	NoteID    uuid.UUID `json:"note_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Content   string    `json:"content"`
	IsEdited  bool      `json:"is_edited"`
	IsDeleted bool      `json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToReplyResponse(r *reply.Reply) ReplyResponse {
	content := r.Content
	if r.IsDeleted {
		content = "This reply was deleted."
	}
	return ReplyResponse{
		ID:        r.ID,
		NoteID:    r.NoteID,
		AuthorID:  r.AuthorID,
		Content:   content,
		IsEdited:  r.IsEdited,
		IsDeleted: r.IsDeleted,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
