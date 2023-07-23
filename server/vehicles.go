package server

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/technopolitica/open-mobility/db"
	"github.com/technopolitica/open-mobility/types"
)

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

		ctx := context.Background()
		conn, err := env.db.Acquire(ctx)
		if err != nil {
			log.Printf("failed to acquire db connection: %s", err)
			w.WriteHeader(500)
			return
		}
		queries := db.New(conn)

		var params []db.RegisterNewVehiclesParams
		response := types.BulkApiResponse[types.Vehicle]{
			Total: len(vehicles),
		}
		for _, vehicle := range vehicles {
			errs := types.ValidateVehicle(vehicle)
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
				ID:                      vehicle.DeviceId,
				ExternalID:              pgtype.Text{String: vehicle.VehicleId, Valid: vehicle.VehicleId != ""},
				Provider:                vehicle.ProviderId,
				DataProvider:            vehicle.DataProviderId,
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
	vehiclesRouter.Get("/{vid:.+}", func(w http.ResponseWriter, r *http.Request) {
		vid := uuid.MustParse(chi.URLParam(r, "vid"))

		ctx := context.Background()
		conn, err := env.db.Acquire(ctx)
		if err != nil {
			log.Printf("failed to acquire db connection: %s", err)
			w.WriteHeader(500)
			return
		}
		queries := db.New(conn)

		vehicle, err := queries.FetchVehicle(ctx, vid)
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
		render.JSON(w, r, types.Vehicle{
			DeviceId:                vehicle.ID,
			VehicleId:               vehicle.ExternalID.String,
			ProviderId:              vehicle.Provider,
			DataProviderId:          vehicle.DataProvider,
			VehicleType:             vehicle.VehicleType,
			PropulsionTypes:         vehicle.PropulsionTypes,
			VehicleAttributes:       vehicle.Attributes,
			AccessibilityAttributes: vehicle.AccessibilityAttributes,
			BatteryCapacity:         int(vehicle.BatteryCapacity.Int32),
			FuelCapacity:            int(vehicle.FuelCapacity.Int32),
			MaximumSpeed:            int(vehicle.MaximumSpeed.Int32),
		})
	})
	return vehiclesRouter
}
