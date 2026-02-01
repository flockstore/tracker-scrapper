package adapter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"tracker-scrapper/internal/features/tracking/domain"

	"github.com/stretchr/testify/assert"
)

func TestServientregaAdapter_GetTrackingHistory(t *testing.T) {
	// Mock JSON Response
	mockJSON := `{
		"ValidationNumber": 4,
		"ValidationResponse": 0,
		"Code": 1,
		"Results": [
			{
				"numeroGuia": "2259200365",
				"fechaEnvio": "31/01/2026 12:51 ",
				"estadoActual": "EN PROCESAMIENTO",
				"movimientos": [
					{
						"estado": "Cerrado",
						"fecha": "31/01/2026 12:51 ",
						"movimiento": "Guia generada",
						"ubicacion": "Bogota (Cundinamarca)",
						"Novedad": "",
						"IdProceso": "1"
					},
					{
						"estado": "Cerrado",
						"fecha": "31/01/2026 17:41 ",
						"movimiento": "Ingreso al centro logistico",
						"ubicacion": "Bogota (Cundinamarca)",
						"Novedad": "",
						"IdProceso": "6"
					},
					{
						"estado": "Cerrado",
						"fecha": "31/01/2026 18:27 ",
						"movimiento": "Salio a ciudad destino",
						"ubicacion": "Bogota (Cundinamarca)",
						"Novedad": "",
						"IdProceso": "12"
					}
				]
			}
		]
	}`

	// Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the request is for the API, return the mock JSON
		if strings.Contains(r.URL.Path, "ControlRastreovalidaciones") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(mockJSON))
			return
		}

		// Otherwise, serve a dummy HTML page that triggers the API call via Fetch
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
				<body>
					<h1>Mock Tracking Page</h1>
					<script>
						// Trigger the API call that the scraper hijacks
						fetch('/api/ControlRastreovalidaciones')
							.then(response => response.json())
							.then(data => console.log(data))
							.catch(error => console.error('Error:', error));
					</script>
				</body>
			</html>
		`) // Use relative path
	}))
	defer ts.Close()

	// Initialize the adapter with the mock server URL
	adapter := NewServientregaAdapter(ts.URL)

	// Call the method
	history, err := adapter.GetTrackingHistory("2259200365")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, history)
	assert.Equal(t, domain.TrackingStatusProcessing, history.GlobalStatus)
	assert.Len(t, history.History, 3)

	// Verify first event
	event1 := history.History[0]
	assert.Equal(t, "Guia generada", event1.Text)
	assert.Equal(t, "Bogota (Cundinamarca)", event1.City)
	assert.Equal(t, "1", event1.Code)

	// Verify date parsing (31/01/2026 12:51)
	expectedTime, _ := time.Parse("02/01/2006 15:04", "31/01/2026 12:51")
	assert.Equal(t, expectedTime, event1.Date)
}
