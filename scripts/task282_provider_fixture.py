#!/usr/bin/env python3
"""Serve controlled USDA/OpenFoodFacts responses for Task 282 acceptance."""

# Implements DESIGN-012 USDAClient/OpenFoodFactsClient controlled HTTP acceptance fixtures.

from __future__ import annotations

import argparse
import json
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlsplit


def usda_food(name: str = "Fixture lentils") -> dict[str, object]:
    return {
        "fdcId": 282001,
        "description": name,
        "servingSize": 100,
        "servingSizeUnit": "g",
        "foodNutrients": [
            {"nutrientName": "Protein", "unitName": "G", "value": 9},
            {"nutrientName": "Carbohydrate, by difference", "unitName": "G", "value": 20},
            {"nutrientName": "Total lipid (fat)", "unitName": "G", "value": 1},
            {"nutrientName": "Sodium, Na", "unitName": "MG", "value": 7},
        ],
        "foodMeasures": [],
    }


def off_product(metadata: bool = False, malformed: bool = False) -> dict[str, object]:
    nutrients: dict[str, object] = {
        "proteins_100g": "bad" if malformed else 4.5,
        "carbohydrates_100g": 11,
        "fat_100g": 2,
        "sodium_100g": 0.02,
    }
    if metadata:
        nutrients.update({"energy_modifier": "~", "proteins_unit": "g", "label": "fixture"})
    return {
        "code": "282002",
        "product_name": "Fixture chickpeas",
        "serving_quantity": 100,
        "serving_quantity_unit": "g",
        "product_quantity": 400,
        "product_quantity_unit": "g",
        "nutriments": nutrients,
    }


class Handler(BaseHTTPRequestHandler):
    """Bounded fixture handler selected solely by the requested search text."""

    server_version = "MealswappTask282Fixture/1"

    def do_GET(self) -> None:  # noqa: N802 - stdlib handler API
        parsed = urlsplit(self.path)
        if parsed.path == "/health":
            self.send_response(204)
            self.end_headers()
            return
        provider = "usda" if parsed.path == "/usda" else "openfoodfacts" if parsed.path == "/openfoodfacts" else ""
        if not provider:
            self.send_error(404)
            return
        parameters = parse_qs(parsed.query)
        query = (parameters.get("query") or parameters.get("search_terms") or [""])[0]
        if query in {"timeout", "cancel"}:
            time.sleep(2)
        if query == "outage" or query == "partial" and provider == "usda":
            self._json(503, {"fixture": "unavailable"})
            return
        if query == "quota" and provider == "usda":
            self._json(
                429,
                {"fixture": "quota"},
                {"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": str(int(time.time()) + 1)},
            )
            return
        if query == "malformed":
            self._json(200, {"not": "a provider search envelope"})
            return
        if provider == "usda":
            if query == "zero":
                self._json(200, {"totalHits": 0, "currentPage": 1, "totalPages": 0, "foods": []})
                return
            food = usda_food()
            if query == "rejected":
                food["fdcId"] = 0
            if query == "optional":
                food["foodMeasures"] = [
                    {"gramWeight": 224, "disseminationText": "1 cup", "amount": None, "measureUnit": None},
                    {"gramWeight": 12, "disseminationText": "variable portion", "amount": None, "measureUnit": None},
                ]
            if query == "zero":
                self._json(200, {"totalHits": 0, "currentPage": 0, "totalPages": 0, "foods": []})
                return
            self._json(200, {"totalHits": 1, "currentPage": 1, "totalPages": 1, "foods": [food]})
            return
        if query == "zero":
            self._json(200, {"count": 0, "page": 1, "page_count": 0, "page_size": 25, "products": []})
            return
        product = off_product(metadata=query == "metadata", malformed=query == "malformed-consumed")
        self._json(200, {"count": 1, "page": 1, "page_count": 1, "page_size": 25, "products": [product]})

    def log_message(self, _format: str, *_args: object) -> None:
        return

    def _json(self, status: int, payload: object, headers: dict[str, str] | None = None) -> None:
        body = json.dumps(payload, separators=(",", ":")).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        for name, value in (headers or {}).items():
            self.send_header(name, value)
        self.end_headers()
        try:
            self.wfile.write(body)
        except BrokenPipeError:
            pass


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--port", type=int, required=True)
    args = parser.parse_args()
    ThreadingHTTPServer(("127.0.0.1", args.port), Handler).serve_forever()


if __name__ == "__main__":
    main()
