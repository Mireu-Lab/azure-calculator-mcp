package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type AzureRetailPriceResponse struct {
	Items        []AzurePriceItem `json:"Items"`
	NextPageLink string           `json:"NextPageLink"`
	Count        int              `json:"Count"`
}

type AzurePriceItem struct {
	CurrencyCode         string  `json:"currencyCode"`
	TierMinimumUnits     float64 `json:"tierMinimumUnits"`
	RetailPrice          float64 `json:"retailPrice"`
	UnitPrice            float64 `json:"unitPrice"`
	ArmPath              string  `json:"armPath"`
	ArmRegionName        string  `json:"armRegionName"`
	Location             string  `json:"location"`
	EffectiveStartDate   string  `json:"effectiveStartDate"`
	MeterId              string  `json:"meterId"`
	MeterName            string  `json:"meterName"`
	ProductId            string  `json:"productId"`
	SkuId                string  `json:"skuId"`
	ProductName          string  `json:"productName"`
	SkuName              string  `json:"skuName"`
	ServiceName          string  `json:"serviceName"`
	ServiceId            string  `json:"serviceId"`
	ServiceFamily        string  `json:"serviceFamily"`
	UnitOfMeasure        string  `json:"unitOfMeasure"`
	Type                 string  `json:"type"`
	IsPrimaryMeterRegion bool    `json:"isPrimaryMeterRegion"`
	ArmSkuName           string  `json:"armSkuName"`
}

func main() {
	outputFilename := "Azure-Comprehensive-Catalog.tsv"
	region := "koreacentral"

	outFile, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("Failed to create catalog file: %v", err)
	}
	defer outFile.Close()

	tsvWriter := csv.NewWriter(outFile)
	tsvWriter.Comma = '\t'
	defer tsvWriter.Flush()

	headers := []string{
		"sku", "service_code", "product_family", "region_code",
		"instance_type", "vcpu", "memory_gib", "storage_type",
		"usage_type", "operation", "purchase_option", "unit",
		"price_per_unit_usd", "description",
	}
	if err := tsvWriter.Write(headers); err != nil {
		log.Fatalf("Failed to write headers: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	filterQuery := fmt.Sprintf("armRegionName eq '%s' and priceType eq 'Consumption'", region)
	apiURL := fmt.Sprintf("https://prices.azure.com/api/retail/prices?$filter=%s", url.QueryEscape(filterQuery))

	log.Printf("Fetching Azure retail prices for region: %s...", region)
	count := 0
	for apiURL != "" && count < 5000 {
		resp, err := client.Get(apiURL)
		if err != nil {
			log.Printf("Request error: %v", err)
			break
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("Unexpected HTTP %d", resp.StatusCode)
			break
		}

		var payload AzureRetailPriceResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			log.Printf("Decode error: %v", err)
			break
		}
		resp.Body.Close()

		for _, item := range payload.Items {
			row := []string{
				item.SkuId,
				item.ServiceName,
				item.ServiceFamily,
				item.ArmRegionName,
				item.ArmSkuName,
				"",
				"",
				item.MeterName,
				item.Type,
				item.ProductName,
				"OnDemand",
				item.UnitOfMeasure,
				fmt.Sprintf("%.6f", item.UnitPrice),
				fmt.Sprintf("%s - %s", item.ProductName, item.MeterName),
			}
			if err := tsvWriter.Write(row); err != nil {
				log.Printf("Write error: %v", err)
			}
			count++
		}
		tsvWriter.Flush()
		apiURL = payload.NextPageLink
		if count >= 3000 {
			break
		}
	}

	log.Printf("Successfully generated Azure catalog (%d records) at %s", count, outputFilename)
}
