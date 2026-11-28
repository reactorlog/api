package api

import "time"

// Site is a plant site as returned by the read API.
type Site struct {
	ID           string    `json:"id"`
	EIAPlantCode *string   `json:"eia_plant_code"`
	Name         string    `json:"name"`
	Timezone     *string   `json:"timezone"`
	Latitude     *float64  `json:"latitude"`
	Longitude    *float64  `json:"longitude"`
	AddressLine1 *string   `json:"address_line_1"`
	AddressLine2 *string   `json:"address_line_2"`
	City         *string   `json:"city"`
	State        *string   `json:"state"`
	Location     *string   `json:"location"`
	Zip          *string   `json:"zip"`
	Phone        *string   `json:"phone"`
	Email        *string   `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ReactorCount int       `json:"reactor_count"`
}

// Reactor is a reactor unit as returned by the read API.
type Reactor struct {
	ID           string    `json:"id"`
	SiteID       string    `json:"site_id"`
	Name         string    `json:"name"`
	UnitNumber   string    `json:"unit_number"`
	DocketNumber string    `json:"docket_number"`
	ReactorType  string    `json:"reactor_type"`
	Design       *string   `json:"design"`
	CapacityMW   *float64  `json:"capacity_mw"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SiteDetail is one site with its reactors.
type SiteDetail struct {
	Site
	Reactors []Reactor `json:"reactors"`
}
