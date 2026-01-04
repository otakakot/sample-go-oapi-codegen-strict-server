package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echomiddleware "github.com/oapi-codegen/echo-middleware"

	"github.com/otakakot/sample-go-oapi-codegen-strict-server/pkg/api"
)

func main() {
	port := cmp.Or(os.Getenv("PORT"), "1323")

	e := echo.New()

	e.Use(middleware.Logger())

	e.Use(middleware.Recover())

	swagger, err := api.GetSwagger()
	if err != nil {
		panic(err)
	}

	// おまじない ref: https://github.com/deepmap/oapi-codegen/blob/master/examples/petstore-expanded/echo/petstore.go#L30-L32
	swagger.Servers = nil

	options := &echomiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: func(ctx context.Context, ai *openapi3filter.AuthenticationInput) error {
				slog.Info(fmt.Sprintf("authentication func: %+v", ai))

				switch ai.SecuritySchemeName {
				case "bearerAuth":
					slog.Info(fmt.Sprintf("bearer token: %s", ai.RequestValidationInput.Request.Header.Get("Authorization")))

					authorization := ai.RequestValidationInput.Request.Header.Get("Authorization")

					authorizations := strings.Split(authorization, " ")

					if len(authorizations) != 2 {
						return fmt.Errorf("invalid authorization: %s", authorization)
					}

					if authorizations[0] != "Bearer" {
						return fmt.Errorf("invalid token: %s", authorization)
					}

					if authorizations[1] != "token" {
						return fmt.Errorf("invalid token: %s", authorizations[1])
					}
				case "cookieAuth":
					cookie, err := ai.RequestValidationInput.Request.Cookie("SESSION")
					if err != nil {
						return fmt.Errorf("cookie not found: %w", err)
					}

					slog.Info(fmt.Sprintf("cookie: %s", cookie.Value))
				default:
					return fmt.Errorf("unknown security scheme: %s", ai.SecuritySchemeName)
				}

				return nil
			},
		},
	}

	e.Use(echomiddleware.OapiRequestValidatorWithOptions(swagger, options))

	e.Use(ErrorHandler)

	s := &Server{
		pets: map[int64]api.Pet{},
	}

	hdl := api.NewStrictHandler(s, nil)

	api.RegisterHandlers(e, hdl)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	go func() {
		slog.Info("start server listen")

		if err := e.Start(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Error("shutting down the server")
		}
	}()

	<-ctx.Done()

	slog.Info("start server shutdown")

	ctx, cansel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cansel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Panic(err)
	}

	slog.Info("done server shutdown")
}

func ErrorHandler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := next(c); err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		return nil
	}
}

var _ api.StrictServerInterface = (*Server)(nil)

type Server struct {
	pets map[int64]api.Pet
}

// CreatePets implements api.StrictServerInterface.
func (srv *Server) CreatePets(
	ctx context.Context,
	request api.CreatePetsRequestObject,
) (api.CreatePetsResponseObject, error) {
	srv.pets[request.Body.Id] = *request.Body

	return api.CreatePets201Response{}, nil
}

// DeleteSession implements api.StrictServerInterface.
func (srv *Server) DeleteSession(
	ctx context.Context,
	request api.DeleteSessionRequestObject,
) (api.DeleteSessionResponseObject, error) {
	cookie := http.Cookie{
		Name:   "SESSSION",
		Value:  "",
		MaxAge: -1,
	}

	return api.DeleteSession200Response{
		Headers: api.DeleteSession200ResponseHeaders{
			SetCookie: cookie.String(),
		},
	}, nil
}

// GetSession implements api.StrictServerInterface.
func (srv *Server) GetSession(
	ctx context.Context,
	request api.GetSessionRequestObject,
) (api.GetSessionResponseObject, error) {
	cookie := http.Cookie{
		Name:  "SESSION",
		Value: uuid.NewString(),
	}

	return api.GetSession200Response{
		Headers: api.GetSession200ResponseHeaders{
			SetCookie: cookie.String(),
		},
	}, nil
}

// ListPets implements api.StrictServerInterface.
func (srv *Server) ListPets(
	ctx context.Context,
	request api.ListPetsRequestObject,
) (api.ListPetsResponseObject, error) {
	pets := make([]api.Pet, 0, *request.Params.Limit)

	for _, pet := range srv.pets {
		pets = append(pets, pet)
	}

	return api.ListPets200JSONResponse{
		Body: pets,
		Headers: api.ListPets200ResponseHeaders{
			XNext: "next",
		},
	}, nil
}

// Redirect implements api.StrictServerInterface.
func (srv *Server) Redirect(
	ctx context.Context,
	request api.RedirectRequestObject,
) (api.RedirectResponseObject, error) {
	return api.Redirect302Response{
		Headers: api.Redirect302ResponseHeaders{
			Location: "https://example.com",
		},
	}, nil
}

// ShowPetById implements api.StrictServerInterface.
func (srv *Server) ShowPetById(
	ctx context.Context,
	request api.ShowPetByIdRequestObject,
) (api.ShowPetByIdResponseObject, error) {
	petID, err := strconv.ParseInt(request.PetId, 10, 64)
	if err != nil {
		return nil, err
	}

	pet, ok := srv.pets[petID]
	if !ok {
		return api.ShowPetById404JSONResponse{}, nil
	}

	return api.ShowPetById200JSONResponse{
		Id:   petID,
		Name: pet.Name,
		Tag:  pet.Tag,
	}, nil
}
