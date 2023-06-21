package models

import "time"

type SearchData struct {
	NextPageToken string `json:"nextPageToken"`
	RegionCode    string `json:"regionCode"`
	PageInfo      struct {
		TotalResults   int `json:"totalResults"`
		ResultsPerPage int `json:"resultsPerPage"`
	} `json:"pageInfo"`
	Items []struct {
		ID struct {
			Kind    string `json:"kind"`
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			PublishedAt          time.Time `json:"publishedAt"`
			ChannelID            string    `json:"channelId"`
			Title                string    `json:"title"`
			Description          string    `json:"description"`
			ChannelTitle         string    `json:"channelTitle"`
			LiveBroadcastContent string    `json:"liveBroadcastContent"`
			PublishTime          time.Time `json:"publishTime"`
		} `json:"snippet"`
	} `json:"items"`
}

type CommentItemData struct {
	ID      string `json:"id"`
	Snippet struct {
		ChannelID       string `json:"channelId"`
		VideoID         string `json:"videoId"`
		TopLevelComment struct {
			ID      string `json:"id"`
			Snippet struct {
				ChannelID             string `json:"channelId"`
				VideoID               string `json:"videoId"`
				TextDisplay           string `json:"textDisplay"`
				TextOriginal          string `json:"textOriginal"`
				AuthorDisplayName     string `json:"authorDisplayName"`
				AuthorProfileImageURL string `json:"authorProfileImageUrl"`
				AuthorChannelURL      string `json:"authorChannelUrl"`
				AuthorChannelID       struct {
					Value string `json:"value"`
				} `json:"authorChannelId"`
				CanRate      bool      `json:"canRate"`
				ViewerRating string    `json:"viewerRating"`
				LikeCount    int       `json:"likeCount"`
				PublishedAt  time.Time `json:"publishedAt"`
				UpdatedAt    time.Time `json:"updatedAt"`
			} `json:"snippet"`
		} `json:"topLevelComment"`
		CanReply        bool `json:"canReply"`
		TotalReplyCount int  `json:"totalReplyCount"`
		IsPublic        bool `json:"isPublic"`
	} `json:"snippet"`
	Replies struct {
		Comments []struct {
			ID      string `json:"id"`
			Snippet struct {
				ChannelID             string `json:"channelId"`
				VideoID               string `json:"videoId"`
				TextDisplay           string `json:"textDisplay"`
				TextOriginal          string `json:"textOriginal"`
				ParentID              string `json:"parentId"`
				AuthorDisplayName     string `json:"authorDisplayName"`
				AuthorProfileImageURL string `json:"authorProfileImageUrl"`
				AuthorChannelURL      string `json:"authorChannelUrl"`
				AuthorChannelID       struct {
					Value string `json:"value"`
				} `json:"authorChannelId"`
				CanRate      bool      `json:"canRate"`
				ViewerRating string    `json:"viewerRating"`
				LikeCount    int       `json:"likeCount"`
				PublishedAt  time.Time `json:"publishedAt"`
				UpdatedAt    time.Time `json:"updatedAt"`
			} `json:"snippet"`
		} `json:"comments"`
	} `json:"replies,omitempty"`
}

type CommentData struct {
	NextPageToken string `json:"nextPageToken"`
	PageInfo      struct {
		TotalResults   int `json:"totalResults"`
		ResultsPerPage int `json:"resultsPerPage"`
	} `json:"pageInfo"`
	Items []*CommentItemData `json:"items"`
}
