package fabdb

type FaBDBSearchResponse struct {
	Data  []Card `json:"data,omitempty"`
	Links struct {
		First string `json:"first"`
		Last  string `json:"last"`
		Prev  string `json:"prev"`
		Next  string `json:"next"`
	} `json:"links"`
	Meta struct {
		CurrentPage int `json:"current_page"`
		From        int `json:"from"`
		LastPage    int `json:"last_page"`
		Links       []struct {
			URL    string `json:"url"`
			Label  string `json:"label"`
			Active bool   `json:"active"`
		} `json:"links"`
		Path    string `json:"path"`
		PerPage string `json:"per_page"`
		To      int    `json:"to"`
		Total   int    `json:"total"`
	} `json:"meta"`
}

type Card struct {
	Identifier     string      `json:"identifier"`
	Name           string      `json:"name"`
	Keywords       []string    `json:"keywords"`
	Text           string      `json:"text"`
	Rarity         string      `json:"rarity"`
	Image          string      `json:"image"`
	SideboardTotal int         `json:"sideboardTotal"`
	Printings      []Printings `json:"printings"`
}

type Printings struct {
	ID       int    `json:"id"`
	Language string `json:"language"`
	Name     string `json:"name"`
	Text     string `json:"text"`
	Flavour  string `json:"flavour"`
	Sku      struct {
		Sku    string `json:"sku"`
		Finish string `json:"finish"`
		Set    struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Released   string `json:"released"`
			Browseable bool   `json:"browseable"`
			Draftable  bool   `json:"draftable"`
		} `json:"set"`
		Number string `json:"number"`
	} `json:"sku"`
	Set     string `json:"set"`
	Rarity  string `json:"rarity"`
	Edition struct {
	}
}

// UniqueSetsFromPrintings is unused for now
func UniqueSetsFromPrintings(printings []Printings) []string {
	sets := make(map[string]interface{})
	for _, p := range printings {
		sets[p.Sku.Set.Name] = ""
	}

	res := []string{}
	for k, _ := range sets {
		res = append(res, k)
	}
	return res
}
