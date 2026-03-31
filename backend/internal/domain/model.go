package domain

import "time"

type Metric struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Value string `json:"value"`
	Trend string `json:"trend"`
}

type DashboardResponse struct {
	UpdatedAt time.Time `json:"updatedAt"`
	Metrics   []Metric  `json:"metrics"`
}

type PersonStats struct {
	TotalPeople     int     `json:"totalPeople"`
	ActivePeople    int     `json:"activePeople"`
	AverageVelocity float64 `json:"averageVelocity"`
}

type Person struct {
	ID               int     `json:"id"`
	FullName         string  `json:"fullName"`
	Role             string  `json:"role"`
	Velocity         float64 `json:"velocity"`
	IsActive         bool    `json:"isActive"`
	TeamID           int     `json:"teamId"`
	TeamLeadID       *int    `json:"teamLeadId,omitempty"`
	BirthDate        *string `json:"birthDate,omitempty"`
	EmploymentDate   *string `json:"employmentDate,omitempty"`
}

type Team struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	LeadID  *int     `json:"leadId,omitempty"`
	Members []Person `json:"members"`
}

type OrgStructureResponse struct {
	UpdatedAt time.Time `json:"updatedAt"`
	Teams     []Team    `json:"teams"`
}

type CreateTeamInput struct {
	Name   string `json:"name"`
	LeadID *int   `json:"leadId"`
}

type CreatePersonInput struct {
	FullName       string  `json:"fullName"`
	Role           string  `json:"role"`
	Velocity       float64 `json:"velocity"`
	IsActive       bool    `json:"isActive"`
	TeamID         int     `json:"teamId"`
	TeamLeadID     *int    `json:"teamLeadId"`
	BirthDate      *string `json:"birthDate"`
	EmploymentDate *string `json:"employmentDate"`
}

type UpdateTeamInput struct {
	Name   string `json:"name"`
	LeadID *int   `json:"leadId"`
}

type UpdatePersonInput struct {
	FullName       string  `json:"fullName"`
	Role           string  `json:"role"`
	Velocity       float64 `json:"velocity"`
	IsActive       bool    `json:"isActive"`
	TeamID         int     `json:"teamId"`
	TeamLeadID     *int    `json:"teamLeadId"`
	BirthDate      *string `json:"birthDate"`
	EmploymentDate *string `json:"employmentDate"`
}
