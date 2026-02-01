# Servientrega Scraper Documentation

This document details the scraping logic, API interaction, and status code mappings for the **Servientrega** courier.

## Overview

- **Courier Code**: `servientrega_co`
- **Scraping Helper**: `go-rod` (Headless Browser)
- **Target URL**: Hardcoded (no config variable)
  - Default: `https://mobile.servientrega.com/WebSitePortal/RastreoEnvioDetalle.html?Guia=%s`

## Scraping Workflow

1.  **Browser Initialization**: Launches a headless Chrome instance.
2.  **Navigation**: Navigates directly to the tracking detail page with the tracking number in the URL.
3.  **Request Hijacking**: Sets up an interception for the internal API endpoint `*/api/ControlRastreovalidaciones`.
4.  **Automatic Trigger**: The page automatically triggers the API call upon load (no interaction needed).
5.  **Response Capture**: Waits for the first API response from the hijacked route.
6.  **Parsing**: Unmarshals the JSON response into the internal `servientregaResponse` struct.

## JSON Response Structure

The scraper expects a JSON response with the following key fields:

```json
{
  "ValidationNumber": 4,
  "ValidationResponse": 0,
  "Code": 1,
  "Results": [
    {
      "numeroGuia": "22000000000",
      "fechaEnvio": "17/01/2026 10:57 ",
      "estadoActual": "ENTREGADO",
      "fechaEstimadaEntrega": "",
      "fechaRealEntrega": "21/01/2026 15:44 ",
      "tipoProducto": "Mercancia premier",
      "movimientos": [
        {
          "estado": "Cerrado",
          "fecha": "17/01/2026 10:57 ",
          "movimiento": "Guia generada",
          "ubicacion": "Bogota (Cundinamarca)",
          "Novedad": "",
          "IdProceso": "1"
        }
      ]
    }
  ]
}
```

## Status Mapping Logic

The global status is derived from the `estadoActual` field. Use the following table to understand the mapping to the Domain `TrackingStatus`.

| Courier Status (`estadoActual`) | Domain Status | Notes |
| :--- | :--- | :--- |
| **ENTREGADO** | `COMPLETED` | Package successfully delivered |
| **ENTREGADO A REMITENTE** | `RETURN` | Package returned to sender |
| **EN PROCESAMIENTO** | `PROCESSING` | Package in transit |
| Other | `PROCESSING` | Default fallback status |

## Event Mapping

Each item in the `movimientos` array is mapped to a `TrackingEvent`:

-   **Date**: Parsed from `fecha` (Expected format: `dd/MM/yyyy HH:mm`, e.g., `31/01/2026 12:51`)
-   **Text**: `movimiento` field, combined with `Novedad` if present (format: `"{movimiento} - {Novedad}"`)
-   **City**: `ubicacion`
-   **Code**: `IdProceso` (string) - Process ID indicating the type of movement (e.g., "1" for "Guia generada", "6" for "Ingreso al centro logistico", "12" for "Salio a ciudad destino")

## Common Process IDs (IdProceso)

| IdProceso | Typical Description |
| :--- | :--- |
| **1** | Guia generada (Guide created) |
| **6** | Ingreso al centro logistico (Arrived at logistics center) |
| **9** | En ruta (In transit) |
| **12** | Salio a ciudad destino (Departed for destination city) |
| **25** | Notificacion de devolucion (Return notification) |

## Implementation Notes

-   **Simple Pattern**: Unlike other adapters, Servientrega uses a very simple scraping pattern - just navigate to the URL and wait for a single API response. No form filling or button clicking required.
-   **No Retry Logic**: The adapter captures the first API response without filtering or validation loops.
-   **Date Format**: Uses non-standard `dd/MM/yyyy HH:mm` format (note the day-first format).
-   **Novedad Field**: The `Novedad` field contains additional information about incidents or special conditions (e.g., "REHUSADO", "M/CIA NO SOLICITADA").

## Debugging

The adapter includes debug logging that outputs:
- `NumeroGuia`: Tracking number from the response
- `EstadoActual`: Current status text
- `Movimientos count`: Number of tracking events in the history

This helps verify that data is being received and parsed correctly.
