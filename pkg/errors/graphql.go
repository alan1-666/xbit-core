package errors

import (
	"encoding/json"
	"net/http"

	"github.com/xbit/xbit-backend/pkg/requestid"
)

const (
	CodeInternal                = "ERR_INTERNAL"
	CodeUpstreamUnavailable     = "ERR_UPSTREAM_UNAVAILABLE"
	CodeUpstreamNotConfigured   = "ERR_UPSTREAM_NOT_CONFIGURED"
	CodeAccessTokenInvalid      = "ErrAccessTokenInvalid"
	CodeUnauthenticated         = "UNAUTHENTICATED"
	CodeValidation              = "ERR_VALIDATION"
	CodeIdempotencyConflict     = "ERR_IDEMPOTENCY_CONFLICT"
	CodeRiskRejected            = "ERR_RISK_REJECTED"
	CodeInsufficientBalance     = "ERR_INSUFFICIENT_BALANCE"
	CodeTransactionSignFailed   = "TRANSACTION_SIGN_FAILED"
	CodeTransactionBroadcasting = "ERR_TRANSACTION_BROADCASTING"
)

type GraphQLResponse struct {
	Errors []GraphQLError `json:"errors"`
}

type GraphQLError struct {
	Message    string          `json:"message"`
	Extensions ErrorExtensions `json:"extensions"`
}

type ErrorExtensions struct {
	Code string         `json:"code"`
	Meta map[string]any `json:"meta,omitempty"`
}

func WriteGraphQLError(w http.ResponseWriter, r *http.Request, status int, code string, message string, meta map[string]any) {
	if meta == nil {
		meta = map[string]any{}
	}
	if traceID := requestid.FromContext(r.Context()); traceID != "" {
		meta["traceId"] = traceID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(GraphQLResponse{
		Errors: []GraphQLError{{
			Message: message,
			Extensions: ErrorExtensions{
				Code: code,
				Meta: meta,
			},
		}},
	})
}
