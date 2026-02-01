# Coordinadora Scraper Documentation

This document details the scraping logic, API interaction, and status code mappings for the **Coordinadora** courier.

## Overview

- **Courier Code**: `coordinadora_co`
- **Helper**: `go-rod`
- **URL**: Configured via `COURIER_COORDINADORA_CO`
  - Default: `https://coordinadora.com/rastreo/rastreo-de-guia/detalle-de-rastreo-de-guia/?guia=%s`

## Scraping Workflow

1.  **Initialization**: Launch headless browser.
2.  **Navigation**: Open URL with the tracking number as a query parameter.
3.  **Hijacking**: Intercept requests to `*/wp-json/rgc/v1/detail_tracking*`.
    - This internal API returns the full history in JSON format.
4.  **Parsing**: Capture response body and unmarshal.

## JSON Structure

Key structure expected from the internal API:

```json
{
  "tracking_number": "...",
  "current_state_text": "...",
  "history": [
    {
      "code": "6",
      "date": "2024-01-03 13:58:00",
      "description": "ENTREGADA",
      "icon_color": "blue"
    },
    {
      "code": "728",
      "date": "...",
      "description": "...",
      "icon_color": "red"
    }
  ]
}
```

## Status Mapping

Status is derived from the `code` field in the `history` array elements.

| Courier Code | Domain Status | Logic |
| :--- | :--- | :--- |
| **6** | `COMPLETED` | Exact match |
| **8** | `RETURN` | Exact match |
| **7**... (e.g., 700, 728) | `INCIDENCE` | Any code starting with "7" |
| Other | `PROCESSING` | Default |

## Event Mapping

-   **Date**: `date` field (Format: `2006-01-02 15:04:05`)
-   **Text**: `description`
-   **City**: Not always explicit in event, logic falls back or leaves empty.
-   **Code**: `code` (string)
