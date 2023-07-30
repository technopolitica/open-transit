package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/technopolitica/open-mobility/db"
	"github.com/technopolitica/open-mobility/types"
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

func presentVehicle(vehicle db.VehicleDenormalized) types.Vehicle {
	return types.Vehicle{
		DeviceID:                vehicle.ID,
		VehicleID:               vehicle.ExternalID.String,
		ProviderID:              vehicle.Provider,
		DataProviderID:          vehicle.DataProvider,
		VehicleType:             vehicle.VehicleType,
		PropulsionTypes:         vehicle.PropulsionTypes,
		VehicleAttributes:       vehicle.Attributes,
		AccessibilityAttributes: vehicle.AccessibilityAttributes,
		BatteryCapacity:         int(vehicle.BatteryCapacity.Int32),
		FuelCapacity:            int(vehicle.FuelCapacity.Int32),
		MaximumSpeed:            int(vehicle.MaximumSpeed.Int32),
	}
}

func NewVehiclesRouter(env *Env) *chi.Mux {
	vehiclesRouter := chi.NewRouter()
	vehiclesRouter.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var vehicles []types.Vehicle
		err := render.DecodeJSON(r.Body, &vehicles)
		if err != nil {
			log.Printf("malformed Vehicle payload: %s", err)
			w.WriteHeader(400)
			render.JSON(w, r, types.ApiError{
				Type:    types.ApiErrorTypeBadParam,
				Details: []string{"vehicles payload is not valid JSON"},
			})
			return
		}

		ctx := r.Context()
		conn, err := env.db.Acquire(ctx)
		if err != nil {
			log.Printf("failed to acquire db connection: %s", err)
			w.WriteHeader(500)
			return
		}
		defer conn.Release()
		queries := db.New(conn)

		var params []db.RegisterNewVehiclesParams
		response := types.BulkApiResponse[types.Vehicle]{
			Total: len(vehicles),
		}
		auth := GetAuthInfo(r)
		for _, vehicle := range vehicles {
			errs := types.ValidateVehicle(vehicle, auth)
			if len(errs) > 0 {
				response.Failures = append(response.Failures, types.FailureDetails[types.Vehicle]{
					Item: vehicle,
					ApiError: types.ApiError{
						Type:    types.ApiErrorTypeBadParam,
						Details: errs,
					},
				})
				continue
			}

			params = append(params, db.RegisterNewVehiclesParams{
				ID:                      vehicle.DeviceID,
				ExternalID:              pgtype.Text{String: vehicle.VehicleID, Valid: vehicle.VehicleID != ""},
				Provider:                vehicle.ProviderID,
				DataProvider:            vehicle.DataProviderID,
				VehicleType:             vehicle.VehicleType,
				Attributes:              vehicle.VehicleAttributes,
				AccessibilityAttributes: vehicle.AccessibilityAttributes,
				PropulsionTypes:         vehicle.PropulsionTypes,
				BatteryCapacity:         pgtype.Int4{Int32: int32(vehicle.BatteryCapacity), Valid: vehicle.BatteryCapacity > 0},
				FuelCapacity:            pgtype.Int4{Int32: int32(vehicle.FuelCapacity), Valid: vehicle.FuelCapacity > 0},
				MaximumSpeed:            pgtype.Int4{Int32: int32(vehicle.MaximumSpeed), Valid: vehicle.MaximumSpeed > 0},
			})
		}
		success, err := queries.RegisterNewVehicles(ctx, params)
		if err != nil {
			log.Printf("failed to write to db: %s", err)
			w.WriteHeader(500)
			return
		}
		// No successful inserts, we should notify the caller.
		if success == 0 {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(201)
		}
		// NOTE: potential integer overflow issue here when processing >2billion vehicles,
		//       though in practice the request size should limited to avoid DOS attacks so we
		//       should never need to worry about this.
		response.Success = int(success)
		render.JSON(w, r, response)
	})
	vehiclesRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		params, errs := parseListVehiclesParams(r)
		if len(errs) > 0 {
			w.WriteHeader(400)
			render.JSON(w, r, types.ApiError{
				Type:    types.ApiErrorTypeBadParam,
				Details: errs,
			})
			return
		}

		ctx := r.Context()
		conn, err := env.db.Acquire(ctx)
		if err != nil {
			log.Printf("failed to acquire db connection: %s", err)
			w.WriteHeader(500)
			return
		}
		defer conn.Release()
		// KLUDGE: can we do a multi-statement batch command here instead of acquiring a transaction?
		tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
		defer tx.Rollback(ctx)
		if err != nil {
			log.Panicf("failed to start transaction: %s", err)
			w.WriteHeader(500)
			return
		}
		queries := db.New(conn).WithTx(tx)

		auth := GetAuthInfo(r)
		count, err := queries.ListVehiclesCount(ctx, auth.ProviderID)
		if err != nil {
			log.Printf("failed to execute count query: %s", err)
			w.WriteHeader(500)
			return
		}
		vehicles, err := queries.ListVehicles(ctx, db.ListVehiclesParams{
			ProviderID: auth.ProviderID,
			Limit:      int32(params.Limit),
			Offset:     int32(params.Offset),
		})
		if err != nil {
			log.Printf("failed execute query: %s", err)
			w.WriteHeader(500)
			return
		}

		tx.Commit(ctx)

		baseURL := types.URL{URL: r.URL}
		first := baseURL.ModifyQuery(func(query *url.Values) {
			query.Set("page[offset]", "0")
		})
		lastOffset := (int(count) / params.Limit) * params.Limit
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
		var prev types.URL
		if hasPrev {
			prev = baseURL.ModifyQuery(func(query *url.Values) {
				query.Set("page[offset]", fmt.Sprint(prevOffset))
			})
		}
		nextOffset := params.Offset + params.Limit
		hasNext := nextOffset <= lastOffset
		var next types.URL
		if hasNext {
			next = baseURL.ModifyQuery(func(query *url.Values) {
				query.Set("page[offset]", fmt.Sprint(nextOffset))
			})
		}

		w.WriteHeader(200)
		presentedVehicles := make([]types.Vehicle, 0, len(vehicles))
		for _, v := range vehicles {
			presentedVehicles = append(presentedVehicles, presentVehicle(v))
		}
		// FIXME: we can't use render.JSON here because the default json.Marshal implementation
		// escapes HTML characters by default (including the ampersand '&'), which breaks
		// the rendering of URLs...
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		err = encoder.Encode(types.PaginatedVehiclesResponse{
			PaginatedResponse: types.PaginatedResponse{
				Version: "2.0.0",
				Links: types.PaginationLinks{
					First: first.String(),
					Last:  last.String(),
					Prev:  prev.String(),
					Next:  next.String(),
				},
			},
			Vehicles: presentedVehicles,
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
			w.WriteHeader(500)
			return
		}
		defer conn.Release()
		queries := db.New(conn)

		auth := GetAuthInfo(r)
		vehicle, err := queries.FetchVehicle(ctx, db.FetchVehicleParams{
			VehicleID:  vid,
			ProviderID: auth.ProviderID,
		})
		if err != nil {
			if err == pgx.ErrNoRows {
				w.WriteHeader(404)
				return
			}
			log.Printf("failed to fetch vehicle: %s", err)
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
		render.JSON(w, r, presentVehicle(vehicle))
	})
	return vehiclesRouter
}
