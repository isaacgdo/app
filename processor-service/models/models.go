package models

import "time"

type VideoItemData struct {
	ID      string `json:"id"`
	Snippet struct {
		PublishedAt          time.Time `json:"publishedAt"`
		ChannelID            string    `json:"channelId"`
		Title                string    `json:"title"`
		Description          string    `json:"description"`
		ChannelTitle         string    `json:"channelTitle"`
		Tags                 []string  `json:"tags"`
		CategoryID           string    `json:"categoryId"`
		LiveBroadcastContent string    `json:"liveBroadcastContent"`
		DefaultLanguage      string    `json:"defaultLanguage"`
		Localized            struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"localized"`
		DefaultAudioLanguage string `json:"defaultAudioLanguage"`
	} `json:"snippet"`
	Statistics struct {
		ViewCount     string `json:"viewCount"`
		LikeCount     string `json:"likeCount"`
		FavoriteCount string `json:"favoriteCount"`
		CommentCount  string `json:"commentCount"`
	} `json:"statistics"`
	TopicDetails struct {
		TopicCategories []string `json:"topicCategories"`
	} `json:"topicDetails"`
}

type VideoListData struct {
	Items    []*VideoItemData `json:"items"`
	PageInfo struct {
		TotalResults   int `json:"totalResults"`
		ResultsPerPage int `json:"resultsPerPage"`
	} `json:"pageInfo"`
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

type CommentOutputData struct {
	ID                string   `json:"id"`
	ChannelID         string   `json:"channelId"`
	ChannelTitle      string   `json:"channelTitle"`
	VideoID           string   `json:"videoId"` // index
	TextDisplay       string   `json:"textDisplay"`
	TextOriginal      string   `json:"textOriginal"`
	AuthorDisplayName string   `json:"authorDisplayName"`
	LikeCount         int      `json:"likeCount"`
	PublishedAt       string   `json:"publishedAt"`
	TotalReplyCount   int      `json:"totalReplyCount"`
	ParentID          string   `json:"parentId"` // is reply
	VideoTitle        string   `json:"videoTitle"`
	VideoViewCount    int      `json:"videoViewCount"`
	VideoLikeCount    int      `json:"videoLikeCount"`
	VideoCommentCount int      `json:"videoCommentCount"`
	VideoTags         []string `json:"videoTags"`
}
