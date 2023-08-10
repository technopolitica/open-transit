package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/technopolitica/open-mobility/internal/db"
	"github.com/technopolitica/open-mobility/internal/domain"
)

const MAX_RESULTS_LIMIT = 20

type ListVehiclesParams struct {
	Limit  int
	Offset int
}

func parseListVehiclesParams(r *http.Request) (params ListVehiclesParams, errs []string) {
	var err error
	offset := r.URL.Query().Get("page[offset]")
	if offset != "" {
		params.Offset, err = strconv.Atoi(offset)
		if err != nil || params.Offset < 0 {
			errs = append(errs, "page[offset]: must be non-negative integer")
		}
	}

	limit := r.URL.Query().Get("page[limit]")
	if limit == "" {
		errs = append(errs, "page[limit]: missing required parameter")
	} else {
		params.Limit, err = strconv.Atoi(limit)
		if err != nil || params.Limit <= 0 {
			errs = append(errs, "page[limit]: must be a positive integer")
		}
		if params.Limit > MAX_RESULTS_LIMIT {
			params.Limit = MAX_RESULTS_LIMIT
			errs = append(errs, fmt.Sprintf("page[limit]: must be less than or equal to %d", MAX_RESULTS_LIMIT))
		}
	}

	return
}

func NewVehiclesRouter(env *Env) *chi.Mux {
	vehiclesRouter := chi.NewRouter()
	vehiclesRouter.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var vehicles []domain.Vehicle
		err := render.DecodeJSON(r.Body, &vehicles)
		if err != nil {
			log.Printf("malformed Vehicle payload: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, domain.ApiError{
				Type:    domain.ApiErrorTypeBadParam,
				Details: []string{"vehicles payload is not valid JSON"},
			})
			return
		}
		defer r.Body.Close()

		ctx := r.Context()
		conn, err := env.db.Acquire(ctx)
		if err != nil {
			log.Printf("failed to acquire db connection: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Release()
		vehicleRepository := db.NewVehicleRepository(conn)

		nServerErrors := 0
		response := domain.BulkApiResponse[domain.Vehicle]{
			Total: len(vehicles),
		}
		auth := GetAuthInfo(r)
		for _, vehicle := range vehicles {
			errs := domain.ValidateVehicle(vehicle)
			if vehicle.ProviderID != auth.ProviderID {
				errs = append(errs, "provider_id: not allowed to register vehicle for another provider")
			}
			if len(errs) > 0 {
				response.Failures = append(response.Failures, domain.FailureDetails[domain.Vehicle]{
					Item: vehicle,
					ApiError: domain.ApiError{
						Type:    domain.ApiErrorTypeBadParam,
						Details: errs,
					},
				})
				continue
			}
			err := vehicleRepository.InsertVehicle(ctx, vehicle)

			if err != nil && errors.Is(err, db.ErrConflict) {
				response.Failures = append(response.Failures, domain.FailureDetails[domain.Vehicle]{
					Item: vehicle,
					ApiError: domain.ApiError{
						Type:    domain.ApiErrorTypeAlreadyRegistered,
						Details: []string{"A vehicle with device_id is already registered"},
					},
				})
				continue
			}

			if err != nil {
				log.Printf("failed to insert vehicle: %s", err)
				response.Failures = append(response.Failures, domain.FailureDetails[domain.Vehicle]{
					Item: vehicle,
					ApiError: domain.ApiError{
						Type:    domain.ApiErrorTypeUnknown,
						Details: []string{"An unknown error has occurred"},
					},
				})
				nServerErrors += 1
				continue
			}

			response.Success += 1
		}

		httpStatus := http.StatusCreated
		// If all of the inserts failed and all of the errors were classified as server errors,
		// return a http.StatusInternalServerError Internal Server Error response to notify the client.
		if nServerErrors == response.Total {
			httpStatus = http.StatusInternalServerError
		} else if response.Success == 0 { // Otherwise if no inserts were successful at least some of them were bad requests
			httpStatus = http.StatusBadRequest
		}
		w.WriteHeader(httpStatus)
		render.JSON(w, r, response)
	})
	vehiclesRouter.Put("/", func(w http.ResponseWriter, r *http.Request) {
		var vehicles []domain.Vehicle
		err := render.DecodeJSON(r.Body, &vehicles)
		if err != nil {
			log.Printf("malformed Vehicle payload: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, domain.ApiError{
				Type:    domain.ApiErrorTypeBadParam,
				Details: []string{"vehicles payload is not valid JSON"},
			})
			return
		}
		defer r.Body.Close()

		ctx := r.Context()
		conn, err := env.db.Acquire(ctx)
		if err != nil {
			log.Printf("failed to acquire db connection: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Release()
		vehicleRepository := db.NewVehicleRepository(conn)

		nServerErrors := 0
		response := domain.BulkApiResponse[domain.Vehicle]{
			Total: len(vehicles),
		}
		auth := GetAuthInfo(r)
		for _, vehicle := range vehicles {
			errs := domain.ValidateVehicle(vehicle)
			if len(errs) > 0 {
				response.Failures = append(response.Failures, domain.FailureDetails[domain.Vehicle]{
					Item: vehicle,
					ApiError: domain.ApiError{
						Type:    domain.ApiErrorTypeBadParam,
						Details: errs,
					},
				})
				continue
			}
			if vehicle.ProviderID != auth.ProviderID {
				response.Failures = append(response.Failures, domain.FailureDetails[domain.Vehicle]{
					Item: vehicle,
					ApiError: domain.ApiError{
						Type:    domain.ApiErrorTypeBadParam,
						Details: []string{"provider_id: does not match user's provider ID"},
					},
				})
				continue
			}

			err := vehicleRepository.UpdateVehicle(ctx, vehicle)

			if err != nil && errors.Is(err, db.ErrNotFound) {
				response.Failures = append(response.Failures, domain.FailureDetails[domain.Vehicle]{
					Item: vehicle,
					ApiError: domain.ApiError{
						Type:    domain.ApiErrorTypeUnregistered,
						Details: []string{},
					},
				})
				continue
			}

			if err != nil {
				log.Printf("failed to update vehicle: %s", err)
				response.Failures = append(response.Failures, domain.FailureDetails[domain.Vehicle]{
					Item: vehicle,
					ApiError: domain.ApiError{
						Type:    domain.ApiErrorTypeUnknown,
						Details: []string{"An unknown error has occurred"},
					},
				})
				nServerErrors += 1
				continue
			}

			response.Success += 1
		}

		httpStatus := http.StatusOK
		// If all of the inserts failed and all of the errors were classified as server errors,
		// return a http.StatusInternalServerError Internal Server Error response to notify the client.
		if nServerErrors == response.Total {
			httpStatus = http.StatusInternalServerError
		} else if response.Success == 0 { // Otherwise if no inserts were successful at least some of them were bad requests
			httpStatus = http.StatusBadRequest
		}
		w.WriteHeader(httpStatus)
		render.JSON(w, r, response)
	})
	vehiclesRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		params, errs := parseListVehiclesParams(r)
		if len(errs) > 0 {
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, domain.ApiError{
				Type:    domain.ApiErrorTypeBadParam,
				Details: errs,
			})
			return
		}

		ctx := r.Context()
		conn, err := env.db.Acquire(ctx)
		if err != nil {
			log.Printf("failed to acquire db connection: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Release()
		// KLUDGE: can we do a multi-statement batch command here instead of acquiring a transaction?
		vehicleRepository := db.NewVehicleRepository(conn)

		auth := GetAuthInfo(r)
		page, err := vehicleRepository.ListVehicles(ctx, domain.ListVehiclesParams{
			ProviderID: auth.ProviderID,
			Limit:      int32(params.Limit),
			Offset:     int32(params.Offset),
		})
		if err != nil {
			log.Printf("failed execute query: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		baseURL := domain.URL{URL: r.URL}
		first := baseURL.ModifyQuery(func(query *url.Values) {
			query.Set("page[offset]", "0")
		})
		lastOffset := (int(page.Total) / params.Limit) * params.Limit
		last := baseURL.ModifyQuery(func(query *url.Values) {
			query.Set("page[offset]", fmt.Sprint(lastOffset))
		})
		prevOffset := params.Offset - params.Limit
		// If we get a nonsensical offset that's greater than the last offset, we'll point
		// the prev link to the last offset.
		if prevOffset > lastOffset {
			prevOffset = lastOffset
		}
		hasPrev := prevOffset >= 0
		var prev domain.URL
		if hasPrev {
			prev = baseURL.ModifyQuery(func(query *url.Values) {
				query.Set("page[offset]", fmt.Sprint(prevOffset))
			})
		}
		nextOffset := params.Offset + params.Limit
		hasNext := nextOffset <= lastOffset
		var next domain.URL
		if hasNext {
			next = baseURL.ModifyQuery(func(query *url.Values) {
				query.Set("page[offset]", fmt.Sprint(nextOffset))
			})
		}

		w.WriteHeader(http.StatusOK)
		// FIXME: we can't use render.JSON here because the default json.Marshal implementation
		// escapes HTML characters by default (including the ampersand '&'), which breaks
		// the rendering of URLs...
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		err = encoder.Encode(domain.PaginatedVehiclesResponse{
			PaginatedResponse: domain.PaginatedResponse{
				Version: "2.0.0",
				Links: domain.PaginationLinks{
					First: first.String(),
					Last:  last.String(),
					Prev:  prev.String(),
					Next:  next.String(),
				},
			},
			Vehicles: page.Items,
		})
		if err != nil {
			panic(err)
		}
	})
	vehiclesRouter.Get("/{vid:.+}", func(w http.ResponseWriter, r *http.Request) {
		vid := uuid.MustParse(chi.URLParam(r, "vid"))

		ctx := r.Context()
		conn, err := env.db.Acquire(ctx)
		if err != nil {
			log.Printf("failed to acquire db connection: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Release()
		vehicleRepository := db.NewVehicleRepository(conn)

		auth := GetAuthInfo(r)
		vehicle, err := vehicleRepository.FetchVehicle(ctx, domain.FetchVehicleParams{
			VehicleID:  vid,
			ProviderID: auth.ProviderID,
		})

		if err != nil && errors.Is(err, db.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err != nil {
			log.Printf("failed to fetch vehicle: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		render.JSON(w, r, vehicle)
	})
	return vehiclesRouter
}
