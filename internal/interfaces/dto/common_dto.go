package dto

import (
	"fmt"

	"github.com/friend-diaries/backend/pkg/apperror"
)

func errRequired(field string) error {
	return apperror.BadRequest(fmt.Sprintf("%s is required", field))
}

func errMinLength(field string, min int) error {
	return apperror.BadRequest(fmt.Sprintf("%s must be at least %d characters", field, min))
}

func errMaxLength(field string, max int) error {
	return apperror.BadRequest(fmt.Sprintf("%s must be at most %d characters", field, max))
}
