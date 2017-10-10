package main

// Tweet holds one tweet
type Tweet struct {
	Text     string `json:"text"`
	Entities `json:"entities"`
}

// Entities is an array of Media objects
type Entities struct {
	Media []Media `json:"media"`
}

// Media objects come in a few different types, but we
// only care about photos
type Media struct {
	Type     string `json:"type"`
	PhotoURL string `json:"media_url"`
}

// GetPhotoURL gets the first photo url if the tweet has one
func (t *Tweet) GetPhotoURL() string {
	if len(t.Media) > 0 && t.Media[0].Type == "photo" {
		return t.Media[0].PhotoURL
	}
	return ""
}
