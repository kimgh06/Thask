package handler

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

type CustomValidator struct {
	validator *validator.Validate
}

func NewValidator() *CustomValidator {
	v := validator.New()
	_ = v.RegisterValidation("slug", validateSlug)
	return &CustomValidator{validator: v}
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

func validateSlug(fl validator.FieldLevel) bool {
	return slugRegex.MatchString(fl.Field().String())
}
