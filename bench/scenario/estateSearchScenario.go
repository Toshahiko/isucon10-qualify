package scenario

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/morikuni/failure"

	"github.com/isucon10-qualify/isucon10-qualify/bench/asset"
	"github.com/isucon10-qualify/isucon10-qualify/bench/client"
	"github.com/isucon10-qualify/isucon10-qualify/bench/fails"
	"github.com/isucon10-qualify/isucon10-qualify/bench/parameter"
)

func estateSearchScenario(ctx context.Context, c *client.Client) error {

	t := time.Now()
	err := c.AccessTopPage(ctx)
	if err != nil {
		fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
		return failure.New(fails.ErrApplication)
	}
	if time.Since(t) > parameter.ThresholdTimeOfAbandonmentPage {
		return failure.New(fails.ErrTimeout)
	}

	t = time.Now()
	err = c.AccessEstateSearchPage(ctx)
	if err != nil {
		fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
		return failure.New(fails.ErrApplication)
	}
	if time.Since(t) > parameter.ThresholdTimeOfAbandonmentPage {
		return failure.New(fails.ErrTimeout)
	}

	// Search Estates with Query
	var er *client.EstatesResponse
	for i := 0; i < parameter.NumOfSearchEstateInScenario; i++ {
		q, err := createRandomEstateSearchQuery()
		if err != nil {
			fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
			return failure.New(fails.ErrApplication)
		}

		t = time.Now()
		_er, err := c.SearchEstatesWithQuery(ctx, q)
		if err != nil {
			fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
			return failure.New(fails.ErrApplication)
		}

		if time.Since(t) > parameter.ThresholdTimeOfAbandonmentPage {
			return failure.New(fails.ErrTimeout)
		}

		if len(_er.Estates) == 0 {
			continue
		}

		if !isEstatesOrderedByRent(_er.Estates) {
			err = failure.New(fails.ErrApplication, failure.Message("GET /api/estate/search: 検索結果が不正です"))
			fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
			return failure.New(fails.ErrApplication)
		}

		er = _er

		numOfPages := int(_er.Count) / parameter.PerPageOfEstateSearch
		if numOfPages == 0 {
			continue
		}

		for j := 0; j < parameter.NumOfCheckEstateSearchPaging; j++ {
			q.Set("page", strconv.Itoa(rand.Intn(numOfPages)))

			t := time.Now()
			_er, err := c.SearchEstatesWithQuery(ctx, q)
			if err != nil {
				fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
				return failure.New(fails.ErrApplication)
			}

			if time.Since(t) > parameter.ThresholdTimeOfAbandonmentPage {
				return failure.New(fails.ErrTimeout)
			}

			if len(_er.Estates) == 0 {
				fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
				return failure.New(fails.ErrApplication)
			}

			if !isEstatesOrderedByRent(_er.Estates) {
				err = failure.New(fails.ErrApplication, failure.Message("GET /api/estate/search: 検索結果が不正です"))
				fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
				return failure.New(fails.ErrApplication)
			}

			er = _er
			numOfPages = int(er.Count) / parameter.PerPageOfEstateSearch
			if numOfPages == 0 {
				break
			}
		}
	}

	if len(er.Estates) == 0 {
		return nil
	}

	// Get Details with ID from previously searched list
	var targetID int64 = -1
	for i := 0; i < parameter.NumOfCheckEstateDetailPage; i++ {
		randomPosition := rand.Intn(len(er.Estates))
		targetID = er.Estates[randomPosition].ID
		t = time.Now()
		e, err := c.AccessEstateDetailPage(ctx, targetID)
		if err != nil {
			fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
			return failure.New(fails.ErrApplication)
		}

		if time.Since(t) > parameter.ThresholdTimeOfAbandonmentPage {
			return failure.New(fails.ErrTimeout)
		}

		estate, err := asset.GetEstateFromID(e.ID)
		if err != nil || !e.Equal(estate) {
			err = failure.New(fails.ErrApplication, failure.Message("GET /api/estate/:id: 物件情報が不正です"))
			fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
			return failure.New(fails.ErrApplication)
		}
	}

	if targetID == -1 {
		return nil
	}

	err = c.RequestEstateDocument(ctx, strconv.FormatInt(targetID, 10))

	if err != nil {
		fails.ErrorsForCheck.Add(err, fails.ErrorOfEstateSearchScenario)
		return failure.New(fails.ErrApplication)
	}

	return nil
}
