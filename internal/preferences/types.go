package preferences

import (
	"fmt"

	"camp-scheduler/internal/api"
)

// AgeGroupPreferenceItem is a single item in a batch replace request for age group preferences.
type AgeGroupPreferenceItem struct {
	AgeGroupID string `json:"age_group_id" binding:"required"`
	Rank       int32  `json:"rank" binding:"required,gt=0"`
}

// CocounselorPreferenceItem is a single item in a batch replace request for co-counselor preferences.
type CocounselorPreferenceItem struct {
	PreferredCounselorID string `json:"preferred_counselor_id" binding:"required"`
	Rank                 int32  `json:"rank" binding:"required,gt=0"`
}

// ActivityPreferenceItem is a single item in a batch replace request for activity preferences.
type ActivityPreferenceItem struct {
	ActivityID string `json:"activity_id" binding:"required"`
	Rank       int32  `json:"rank" binding:"required,gt=0"`
}

// CamperFriendPreferenceItem is a single item in a batch replace request for camper friend preferences.
type CamperFriendPreferenceItem struct {
	PreferredCamperID string `json:"preferred_camper_id" binding:"required"`
	Rank              int32  `json:"rank" binding:"required,gt=0"`
}

// validateRanks checks that ranks are sequential starting from 1 with no gaps.
func validateRanks(n int, rankAt func(i int) int32) error {
	for i := 0; i < n; i++ {
		expected := int32(i + 1)
		if rankAt(i) != expected {
			return api.BadInput(fmt.Sprintf("ranks must be sequential starting from 1, got %d at position %d", rankAt(i), i+1))
		}
	}
	return nil
}
