package adapter

import (
	"encoding/json"
	"testing"

	"tracker-scrapper/internal/features/tracking/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCoordinadoraAdapter_mapResponseToDomain_Success verifies success mapping (Code 6).
func TestCoordinadoraAdapter_mapResponseToDomain_Success(t *testing.T) {
	jsonContent := `{
    "tracking_number": "04333004120",
    "history": [
        {
            "code": "2",
            "date": "2023-12-28 10:50:44",
            "description": "EN TERMINAL ORIGEN"
        },
        {
            "code": "6",
            "date": "2024-01-03 13:58:00",
            "description": "ENTREGADA"
        }
    ]
}`
	var resp coordinadoraResponse
	err := json.Unmarshal([]byte(jsonContent), &resp)
	require.NoError(t, err)

	adapter := &CoordinadoraAdapter{
		logger: zap.NewNop(),
	}
	history, err := adapter.mapResponseToDomain(resp)

	require.NoError(t, err)
	assert.Equal(t, domain.TrackingStatusCompleted, history.GlobalStatus)
	require.Len(t, history.History, 2)
	assert.Equal(t, "6", history.History[1].Code)
}

// TestCoordinadoraAdapter_mapResponseToDomain_Return verifies return mapping (Code 8).
func TestCoordinadoraAdapter_mapResponseToDomain_Return(t *testing.T) {
	jsonContent := `{
    "history": [
        {
            "code": "8",
            "date": "2023-10-10 18:05:12",
            "description": "CERRADO POR INCIDENCIA"
        }
    ]
}`
	var resp coordinadoraResponse
	err := json.Unmarshal([]byte(jsonContent), &resp)
	require.NoError(t, err)

	adapter := &CoordinadoraAdapter{
		logger: zap.NewNop(),
	}
	history, err := adapter.mapResponseToDomain(resp)

	require.NoError(t, err)
	assert.Equal(t, domain.TrackingStatusReturn, history.GlobalStatus)
}

// TestCoordinadoraAdapter_mapResponseToDomain_Incidence verifies incidence mapping (Code 7xx).
func TestCoordinadoraAdapter_mapResponseToDomain_Incidence(t *testing.T) {
	jsonContent := `{
    "history": [
        {
            "code": "728",
            "date": "2023-12-30 08:39:40",
            "description": "Destinatario no cancela"
        }
    ]
}`
	var resp coordinadoraResponse
	err := json.Unmarshal([]byte(jsonContent), &resp)
	require.NoError(t, err)

	adapter := &CoordinadoraAdapter{
		logger: zap.NewNop(),
	}
	history, err := adapter.mapResponseToDomain(resp)

	require.NoError(t, err)
	assert.Equal(t, domain.TrackingStatusIncidence, history.GlobalStatus)
	assert.Equal(t, "728", history.History[0].Code)
}

// TestCoordinadoraAdapter_mapResponseToDomain_IncidenceVariations verifies 700 and 701.
func TestCoordinadoraAdapter_mapResponseToDomain_IncidenceVariations(t *testing.T) {
	jsonContent := `{
    "history": [
        {
            "code": "700",
            "date": "2023-12-30 08:39:40",
            "description": "Incidence 700"
        },
        {
            "code": "701",
            "date": "2023-12-31 08:39:40",
            "description": "Incidence 701"
        }
    ]
}`
	var resp coordinadoraResponse
	err := json.Unmarshal([]byte(jsonContent), &resp)
	require.NoError(t, err)

	adapter := &CoordinadoraAdapter{
		logger: zap.NewNop(),
	}
	history, err := adapter.mapResponseToDomain(resp)

	require.NoError(t, err)
	assert.Equal(t, domain.TrackingStatusIncidence, history.GlobalStatus)
	require.Len(t, history.History, 2)
	assert.Equal(t, "700", history.History[0].Code)
	assert.Equal(t, "701", history.History[1].Code)
}
