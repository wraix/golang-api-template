package endpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/charmixer/golang-api-template/endpoint/problem"

	"github.com/hetiansu5/urlquery"
	"go.opentelemetry.io/otel"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var (
	validate *validator.Validate
	locale   string
	trans    ut.Translator
)

func init() {
	validate = validator.New()

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {
			return ""
		}

		return name
	})

	locale = "en"

	enTranslator := en.New()
	uni := ut.New(enTranslator, enTranslator)

	trans, _ = uni.GetTranslator(locale)
	en_translations.RegisterDefaultTranslations(validate, trans)
}

func WithRequestValidation(ctx context.Context, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "request-validation")
	defer span.End()

	err := validate.Struct(i)
	if err == nil {
		// No validation error, continue
		return nil
	}

	prob := problem.NewValidationProblem(http.StatusBadRequest)
	for _, verr := range err.(validator.ValidationErrors) {
		prob.Add(verr.Field(), verr.Translate(trans))
	}

	return prob
}

func WithResponseValidation(ctx context.Context, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "response-validation")
	defer span.End()

	err := validate.Struct(i)
	if err == nil {
		// No validation error, continue
		return nil
	}

	prob := problem.NewValidationProblem(http.StatusInternalServerError)
	for _, verr := range err.(validator.ValidationErrors) {
		prob.Add(verr.Field(), verr.Translate(trans))
	}

	return prob
}

func WithJsonRequestParser(ctx context.Context, r *http.Request, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "request-parser")
	defer span.End()

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&i)
	if err != nil {
		return problem.New(http.StatusBadRequest).WithErr(err)
	}

	return nil
}

func WithRequestQueryParser(ctx context.Context, r *http.Request, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "request-parser-query")
	defer span.End()

	err := urlquery.Unmarshal([]byte(r.URL.RawQuery), i)
	if err != nil {
		return problem.New(http.StatusBadRequest).WithErr(err)
	}

	return nil
}

func WithJsonResponseWriter(ctx context.Context, w http.ResponseWriter, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "json-response-writer")
	defer span.End()

	d, err := json.Marshal(i)
	if err != nil {
		return problem.New(http.StatusInternalServerError).WithErr(err)
	}
	w.Write(d)

	return nil
}

func WithYamlResponseWriter(ctx context.Context, w http.ResponseWriter, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "yaml-response-writer")
	defer span.End()

	d, err := yaml.Marshal(i)
	if err != nil {
		return problem.New(http.StatusInternalServerError).WithErr(err)
	}
	w.Write(d)

	return nil
}

func WithResponseWriter(ctx context.Context, w http.ResponseWriter, tp string, i interface{}) error {
	switch tp {
	case "json":
		return WithJsonResponseWriter(ctx, w, i)
	case "yaml":
		return WithYamlResponseWriter(ctx, w, i)
	default:
		panic(fmt.Sprintf("Unknown response type given, %s", tp))
	}
}
