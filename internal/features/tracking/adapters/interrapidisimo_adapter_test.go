package adapter

import (
	"encoding/json"
	"testing"
	"time"

	"tracker-scrapper/internal/features/tracking/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestInterrapidisimoAdapter_mapResponseToDomain_Success verifies successful delivery parsing.
func TestInterrapidisimoAdapter_mapResponseToDomain_Success(t *testing.T) {
	// JSON content from success.json
	jsonContent := `{
    "TrazaGuia": {
        "IdEstadoGuia": 16,
        "DescripcionEstadoGuia": "Archivada",
        "FechaGrabacion": "2025-05-10T13:06:23.02"
    },
    "EstadosGuia": [
        {
            "EstadoGuia": {
                "IdEstadoGuia": 1,
                "DescripcionEstadoGuia": "Recibimos tú envío",
                "Ciudad": "BOGOTA\\CUND\\COL",
                "FechaGrabacion": "2025-04-30T18:53:15.917"
            }
        },
        {
            "EstadoGuia": {
                "IdEstadoGuia": 11,
                "DescripcionEstadoGuia": "Tú envío fue entregado",
                "Ciudad": "BARRANCABERMEJA\\SANT\\COL",
                "FechaGrabacion": "2025-05-10T13:06:22.83"
            }
        }
    ],
    "Success": true,
    "Message": "Consulta exitosa"
}`

	var resp interResponse
	err := json.Unmarshal([]byte(jsonContent), &resp)
	require.NoError(t, err)

	adapter := &InterrapidisimoAdapter{
		logger: zap.NewNop(),
	}
	history, err := adapter.mapResponseToDomain(resp)

	require.NoError(t, err)
	assert.Equal(t, domain.TrackingStatusCompleted, history.GlobalStatus)
	require.Len(t, history.History, 2)

	// Verify first event
	assert.Equal(t, "1", history.History[0].Code)
	assert.Equal(t, "Recibimos tú envío", history.History[0].Text)
	assert.Equal(t, "BOGOTA\\CUND\\COL", history.History[0].City)

	// Verify date parsing (approximate check due to time zones if not handled strictly)
	expectedDate, _ := time.Parse("2006-01-02T15:04:05", "2025-04-30T18:53:15.917")
	assert.WithinDuration(t, expectedDate, history.History[0].Date, time.Second)

	// Verify delivery event
	assert.Equal(t, "11", history.History[1].Code)
	assert.Equal(t, "Tú envío fue entregado", history.History[1].Text)
}

// TestInterrapidisimoAdapter_mapResponseToDomain_Return verifies return status parsing.
func TestInterrapidisimoAdapter_mapResponseToDomain_Return(t *testing.T) {
	// JSON content from return.json
	jsonContent := `{
    "EstadosGuia": [
        {
            "EstadoGuia": {
                "IdEstadoGuia": 1,
                "DescripcionEstadoGuia": "Recibimos tú envío",
                "Ciudad": "BOGOTA",
                "FechaGrabacion": "2025-10-24T16:27:15.273"
            }
        },
        {
            "EstadoGuia": {
                "IdEstadoGuia": 10,
                "DescripcionEstadoGuia": "Tu envío Fue devuelto",
                "Ciudad": "MEDELLIN",
                "FechaGrabacion": "2025-10-28T09:36:37.19"
            }
        }
    ],
    "Success": true,
    "Message": "Consulta exitosa"
}`

	var resp interResponse
	err := json.Unmarshal([]byte(jsonContent), &resp)
	require.NoError(t, err)

	adapter := &InterrapidisimoAdapter{
		logger: zap.NewNop(),
	}
	history, err := adapter.mapResponseToDomain(resp)

	require.NoError(t, err)
	assert.Equal(t, domain.TrackingStatusReturn, history.GlobalStatus)
	require.Len(t, history.History, 2)

	// Verify return event
	assert.Equal(t, "10", history.History[1].Code)
	assert.Equal(t, "Tu envío Fue devuelto", history.History[1].Text)
}
