package utils

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

//errors

func InitValidator() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation(
			"shortcode",
			customCodeValidator,
		)
	}
}

// customCodeValidator validator rules
func customCodeValidator(f validator.FieldLevel) bool {
	code := f.Field().String()
	match, _ := regexp.MatchString(`^[a-zA-Z0-9_-]{4,20}$`, code)
	return match
}

func ValidateErrors(err error) map[string]string {
	fields := map[string]string{}

	for _, e := range err.(validator.ValidationErrors) {
		switch e.Tag() {
		case "required":
			fields[e.Field()] = "is required"
		case "url":
			fields[e.Field()] = "give valid URL"
		case "min":
			fields[e.Field()] = "too short"
		case "max":
			fields[e.Field()] = "too long"
		default:
			fields[e.Field()] = "invalid"
		}
	}

	return fields
}
