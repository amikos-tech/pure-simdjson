package purejson

type benchTwitterRow struct {
	SearchMetadata benchTwitterSearchMetadata `json:"search_metadata"`
	Statuses       []benchTwitterStatus       `json:"statuses"`
}

type benchTwitterSearchMetadata struct {
	Count       int64  `json:"count"`
	MaxID       int64  `json:"max_id"`
	NextResults string `json:"next_results"`
	Query       string `json:"query"`
}

type benchTwitterStatus struct {
	CreatedAt     string               `json:"created_at"`
	Entities      benchTwitterEntities `json:"entities"`
	FavoriteCount int64                `json:"favorite_count"`
	ID            int64                `json:"id"`
	RetweetCount  int64                `json:"retweet_count"`
	Text          string               `json:"text"`
	User          benchTwitterUser     `json:"user"`
}

type benchTwitterEntities struct {
	Hashtags     []benchTwitterHashtag     `json:"hashtags"`
	UserMentions []benchTwitterUserMention `json:"user_mentions"`
}

type benchTwitterHashtag struct {
	Indices []int  `json:"indices"`
	Text    string `json:"text"`
}

type benchTwitterUserMention struct {
	ID         int64  `json:"id"`
	IDStr      string `json:"id_str"`
	Indices    []int  `json:"indices"`
	Name       string `json:"name"`
	ScreenName string `json:"screen_name"`
}

type benchTwitterUser struct {
	ID         int64  `json:"id"`
	ScreenName string `json:"screen_name"`
	Verified   bool   `json:"verified"`
}

type benchCITMRow struct {
	AreaNames                map[string]string         `json:"areaNames"`
	AudienceSubCategoryNames map[string]string         `json:"audienceSubCategoryNames"`
	BlockNames               map[string]string         `json:"blockNames"`
	Events                   map[string]benchCITMEvent `json:"events"`
	Performances             []benchCITMPerformance    `json:"performances"`
	SeatCategoryNames        map[string]string         `json:"seatCategoryNames"`
	SubTopicNames            map[string]string         `json:"subTopicNames"`
	SubjectNames             map[string]string         `json:"subjectNames"`
	TopicNames               map[string]string         `json:"topicNames"`
	TopicSubTopics           map[string][]int64        `json:"topicSubTopics"`
	VenueNames               map[string]string         `json:"venueNames"`
}

type benchCITMEvent struct {
	Description string  `json:"description"`
	ID          int64   `json:"id"`
	Logo        string  `json:"logo"`
	Name        string  `json:"name"`
	SubTopicIDs []int64 `json:"subTopicIds"`
	SubjectCode string  `json:"subjectCode"`
	Subtitle    string  `json:"subtitle"`
	TopicIDs    []int64 `json:"topicIds"`
}

type benchCITMPerformance struct {
	EventID        int64                   `json:"eventId"`
	ID             int64                   `json:"id"`
	Logo           string                  `json:"logo"`
	Name           string                  `json:"name"`
	Prices         []benchCITMPrice        `json:"prices"`
	SeatCategories []benchCITMSeatCategory `json:"seatCategories"`
	SeatMapImage   string                  `json:"seatMapImage"`
	Start          int64                   `json:"start"`
	VenueCode      string                  `json:"venueCode"`
}

type benchCITMPrice struct {
	Amount                int64 `json:"amount"`
	AudienceSubCategoryID int64 `json:"audienceSubCategoryId"`
	SeatCategoryID        int64 `json:"seatCategoryId"`
}

type benchCITMSeatCategory struct {
	Areas          []benchCITMArea `json:"areas"`
	SeatCategoryID int64           `json:"seatCategoryId"`
}

type benchCITMArea struct {
	AreaID   int64   `json:"areaId"`
	BlockIDs []int64 `json:"blockIds"`
}

type benchCanadaRow struct {
	Features []benchCanadaFeature `json:"features"`
	Type     string               `json:"type"`
}

type benchCanadaFeature struct {
	Geometry   benchCanadaGeometry   `json:"geometry"`
	Properties benchCanadaProperties `json:"properties"`
	Type       string                `json:"type"`
}

type benchCanadaGeometry struct {
	Coordinates [][][]float64 `json:"coordinates"`
	Type        string        `json:"type"`
}

type benchCanadaProperties struct {
	Name string `json:"name"`
}
