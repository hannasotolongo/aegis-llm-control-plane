package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/scheduler"
)

type Server struct {
	store     cluster.StateStore
	scheduler *scheduler.Service
	mux       *http.ServeMux
}

func NewServer(
	store cluster.StateStore,
	schedulerService *scheduler.Service,
) *Server {
	s := &Server{
		store:     store,
		scheduler: schedulerService,
		mux:       http.NewServeMux(),
	}

	s.registerRoutes()

	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)

	s.mux.HandleFunc("/v1/workers", s.handleWorkers)
	s.mux.HandleFunc("/v1/workers/", s.handleWorkerByID)

	s.mux.HandleFunc("/v1/workloads", s.handleWorkloads)
	s.mux.HandleFunc("/v1/workloads/", s.handleWorkloadByID)

	s.mux.HandleFunc("/v1/schedule", s.handleSchedule)
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	placed, err := s.scheduler.SchedulePending(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, placed)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (s *Server) handleWorkers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		workers, err := s.store.ListWorkers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, workers)

	case http.MethodPost:
		var worker cluster.Worker

		if err := json.NewDecoder(r.Body).Decode(&worker); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := s.store.RegisterWorker(r.Context(), worker); err != nil {
			switch {
			case errors.Is(err, cluster.ErrWorkerAlreadyExists):
				writeError(w, http.StatusConflict, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		writeJSON(w, http.StatusCreated, worker)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleWorkerByID(w http.ResponseWriter, r *http.Request) {
	workerID := strings.TrimPrefix(r.URL.Path, "/v1/workers/")

	if workerID == "" {
		writeError(w, http.StatusBadRequest, "worker id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		worker, err := s.store.GetWorker(r.Context(), workerID)
		if err != nil {
			if errors.Is(err, cluster.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}

			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, worker)

	case http.MethodPut:
		var worker cluster.Worker

		if err := json.NewDecoder(r.Body).Decode(&worker); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		worker.ID = workerID

		if err := s.store.UpdateWorker(r.Context(), worker); err != nil {
			if errors.Is(err, cluster.ErrWorkerNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}

			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, worker)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleWorkloads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		workloads, err := s.store.ListWorkloads(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, workloads)

	case http.MethodPost:
		var workload cluster.Workload

		if err := json.NewDecoder(r.Body).Decode(&workload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := s.store.CreateWorkload(r.Context(), workload); err != nil {
			switch {
			case errors.Is(err, cluster.ErrWorkloadAlreadyExists):
				writeError(w, http.StatusConflict, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		writeJSON(w, http.StatusCreated, workload)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleWorkloadByID(w http.ResponseWriter, r *http.Request) {
	workloadID := strings.TrimPrefix(r.URL.Path, "/v1/workloads/")

	if strings.HasSuffix(workloadID, "/place") {
		workloadID = strings.TrimSuffix(workloadID, "/place")
		handlePlacementPath(s, w, r, workloadID)
		return
	}

	if workloadID == "" {
		writeError(w, http.StatusBadRequest, "workload id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		workload, err := s.store.GetWorkload(r.Context(), workloadID)
		if err != nil {
			if errors.Is(err, cluster.ErrWorkloadNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}

			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, workload)

	case http.MethodPut:
		var workload cluster.Workload

		if err := json.NewDecoder(r.Body).Decode(&workload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		workload.ID = workloadID

		if err := s.store.UpdateWorkload(r.Context(), workload); err != nil {
			if errors.Is(err, cluster.ErrWorkloadNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}

			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, workload)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handlePlacementPath(s *Server, w http.ResponseWriter, r *http.Request, workloadID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	placed, err := s.scheduler.PlaceWorkload(
		r.Context(),
		workloadID,
	)
	if err != nil {
		switch {
		case errors.Is(err, cluster.ErrWorkloadNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, scheduler.ErrNoEligibleWorker):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, placed)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
