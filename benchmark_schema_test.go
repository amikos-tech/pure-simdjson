package purejson

import (
	"encoding/json"
	"testing"
)

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
	Favorited     bool                 `json:"favorited"`
	ID            int64                `json:"id"`
	RetweetCount  int64                `json:"retweet_count"`
	Retweeted     bool                 `json:"retweeted"`
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
	Name       string `json:"name"`
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

func TestBenchmarkSchemaDecodesFixtures(t *testing.T) {
	for _, fixtureName := range []string{benchmarkFixtureTwitter, benchmarkFixtureCITM, benchmarkFixtureCanada} {
		value, err := benchmarkDecodeSharedSchema(json.Unmarshal, fixtureName, loadBenchmarkFixture(t, fixtureName))
		if err != nil {
			t.Fatalf("benchmarkDecodeSharedSchema(%s): %v", fixtureName, err)
		}

		switch row := value.(type) {
		case benchTwitterRow:
			if row.SearchMetadata.MaxID == 0 || row.SearchMetadata.Query == "" || len(row.Statuses) == 0 {
				t.Fatalf("twitter schema decoded sparse root: %+v", row.SearchMetadata)
			}
			if row.Statuses[0].ID == 0 || row.Statuses[0].User.ID == 0 || row.Statuses[0].User.ScreenName == "" {
				t.Fatalf("twitter schema decoded sparse first status: %+v", row.Statuses[0])
			}
		case benchCITMRow:
			if len(row.AreaNames) == 0 || len(row.Events) == 0 || len(row.Performances) == 0 {
				t.Fatalf("CITM schema decoded sparse root: areas=%d events=%d performances=%d", len(row.AreaNames), len(row.Events), len(row.Performances))
			}
			if row.Performances[0].ID == 0 || row.Performances[0].EventID == 0 || len(row.Performances[0].Prices) == 0 {
				t.Fatalf("CITM schema decoded sparse first performance: %+v", row.Performances[0])
			}
		case benchCanadaRow:
			if row.Type == "" || len(row.Features) == 0 {
				t.Fatalf("Canada schema decoded sparse root: type=%q features=%d", row.Type, len(row.Features))
			}
			if row.Features[0].Type == "" || row.Features[0].Geometry.Type == "" || len(row.Features[0].Geometry.Coordinates) == 0 {
				t.Fatalf("Canada schema decoded sparse first feature: %+v", row.Features[0])
			}
		default:
			t.Fatalf("benchmarkDecodeSharedSchema(%s) returned %T", fixtureName, value)
		}
	}
}
