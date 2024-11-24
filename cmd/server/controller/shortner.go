package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-batteries/shortner/app/models"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

const (
	AcceptTypeJSON = "application/json"
	AcceptTypeHTML = "text/html"
)

type URLShortner struct {
	urlRepo *models.URLRepo
}

func NewURLShortnerCtrl(urlRepo *models.URLRepo) *URLShortner {
	return &URLShortner{
		urlRepo: urlRepo,
	}
}

func (ctrl *URLShortner) Get(c echo.Context) error {
	req := c.Request()
	accept := req.Header.Get("Accept")
	shortKey := strings.TrimSpace(c.Param("shortKey"))

	expectsJSONResp := strings.EqualFold(accept, AcceptTypeJSON)

	if shortKey == "" {
		errCode := http.StatusBadRequest

		if expectsJSONResp {
			return c.JSON(errCode, `{"success": false, "error": "empty_url"}`)
		}

		return c.HTML(errCode, `<html><body>Fuck off</body></html>`)
	}

	url, err := ctrl.urlRepo.Find(req.Context(), shortKey)
	if url != nil && url.Link == nil {
		err = errors.New("unassigned")
	}

	if err != nil {
		log.Error().Err(err).Msgf("failed to get url from short key %s", shortKey)

		if expectsJSONResp {
			return c.JSON(http.StatusNotFound, `{"success": false, "error": "not_found"}`)
		}

		return c.HTML(http.StatusNotFound, `<html><body>Not Found</body></html>`)
	}

	if expectsJSONResp {
		return c.JSON(http.StatusOK, fmt.Sprintf(`{"success": true, "url": "%s"}`, url.Link))
	}

	return c.Redirect(http.StatusTemporaryRedirect, *url.Link)
}
