package audio

import (
	"context"
	"fmt"
	"github.com/sosodev/duration"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"mrs/internal/dto"
	"strings"
)

var (
	typeQuery      = "video"
	id             = "id"
	snippet        = "snippet"
	contentDetails = "contentDetails"
)

type ServiceAudio struct {
	youtube *youtube.Service
	limit   int64
}

func NewServiceAudio(token string, limit int64) (*ServiceAudio, error) {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(token))
	if err != nil {
		return nil, err
	}

	return &ServiceAudio{youtube: service, limit: limit}, nil
}

func (s *ServiceAudio) GetListVideo(ctx context.Context, query string) ([]*dto.Video, error) {
	// строим запрос на получение url и названия видео
	searchCall := s.youtube.Search.List([]string{id, snippet}).Q(query).Type(typeQuery).MaxResults(s.limit).Context(ctx)

	// делаем запрос
	response, err := searchCall.Do()
	if err != nil {
		return nil, err
	}

	// делаем слайс id для получения продолжительности видео
	ids := make([]string, len(response.Items))
	for i, item := range response.Items {
		ids[i] = item.Id.VideoId
	}

	// делаем запрос сразу по всем
	videoCall := s.youtube.Videos.List([]string{contentDetails}).Id(strings.Join(ids, ",")).Context(ctx)
	respVideo, err := videoCall.Do()
	if err != nil {
		return nil, err
	}

	durationsMap := make(map[string]int64)

	// раскидываю таймы по id
	for _, item := range respVideo.Items {
		d, err := duration.Parse(item.ContentDetails.Duration)
		if err != nil {
			return nil, err
		}

		durationsMap[item.Id] = durationOnSecond(d)
	}

	result := make([]*dto.Video, len(response.Items))

	// формируем ответ
	for i, item := range response.Items {
		res := &dto.Video{
			URL:      fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.Id.VideoId),
			Title:    item.Snippet.Title,
			Duration: durationsMap[item.Id.VideoId],
		}
		result[i] = res
	}
	return result, nil
}

// вспомогательная функция для преобразования в int64
func durationOnSecond(d *duration.Duration) int64 {
	return int64(d.Seconds) + (int64(d.Minutes) * 60) + (int64(d.Hours) * 3600)
}
