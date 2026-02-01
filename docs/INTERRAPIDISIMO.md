# Interrapidisimo Scraper Documentation

This document details the scraping logic, API interaction, and status code mappings for the **Interrapidisimo** courier.

## Overview

- **Courier Code**: `interrapidisimo_co`
- **Scraping Helper**: `go-rod` (Headless Browser)
- **Target URL**: Configured via `COURIER_INTERRAPIDISIMO_CO`
  - Default: `https://www3.interrapidisimo.com/SiguetuEnvio/shipment`

## Scraping Workflow

1.  **Browser Initialization**: Launches a headless Chrome instance.
2.  **Navigation**: Navigates to the shipment tracking page.
3.  **Request Hijacking**: Sets up an interception for the internal API endpoint `*/ObtenerRastreoGuiasClientePost`.
4.  **Interaction**:
    - Inputs the tracking number into `#inputGuide`.
    - Clicks the search button (`.search-button`).
5.  **Response Capture**: Waits for the XHR response from the hijacked route.
6.  **Parsing**: Unmarshals the JSON response into the internal `interResponse` struct.

## JSON Response Structure

The scraper expects a JSON response with the following key fields:

```json
{
  "EstadosGuia": [
    {
      "EstadoGuia": {
        "IdEstadoGuia": 1,
        "DescripcionEstadoGuia": "Recibimos tú envío",
        "Ciudad": "BOGOTA",
        "FechaGrabacion": "2025-04-30T18:53:15.917"
      }
    }
  ],
  "Success": true,
  "Message": "Consulta exitosa"
}
```

## Status Mapping Logic

The generic global status is derived from the `IdEstadoGuia` codes found in the history. Use the following table to understand the mapping to the Domain `TrackingStatus`.

| Courier Code (`IdEstadoGuia`) | Description | Domain Status | Notes |
| :--- | :--- | :--- | :--- |
| **11** | "Tú envío fue entregado" | `COMPLETED` | Marks shipment as Delivered=true |
| **10** | "Tu envío Fue devuelto" | `RETURN` | Indicates a return flow |
| **7** | Various incidence types | `INCIDENCE` | Mapped to Incidence status |
| **1, 2, 3...** | various transit states | `PROCESSING` | Default state |

## Event Mapping

Each item in `EstadosGuia` is mapped to a `TrackingEvent`:

-   **Date**: Parsed from `FechaGrabacion` (Expected format: RFC3339-like `2006-01-02T15:04:05`)
-   **Text**: `DescripcionEstadoGuia`
-   **City**: `Ciudad`
-   **Code**: `IdEstadoGuia` (converted to string) as the raw provider code.
