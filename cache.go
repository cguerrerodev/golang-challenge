package sample1

import (
	"fmt"
	"sync"
	"time"
)

// PriceService is a service that we can use to get prices for the items
// Calls to this service are expensive (they take time)
type PriceService interface {
	GetPriceFor(itemCode string) (float64, error)
}

// TransparentCache is a cache that wraps the actual service
// The cache will remember prices we ask for, so that we don't have to wait on every call
// Cache should only return a price if it is not older than "maxAge", so that we don't get stale prices

type TransparentCache struct {
	actualPriceService PriceService
	maxAge             time.Duration
	prices             map[string]priceInformation
}

type priceInformation struct {
	lastReading time.Time
	value       float64
}

type result struct {
	price float64
	err   error
}

func NewTransparentCache(actualPriceService PriceService, maxAge time.Duration) *TransparentCache {
	return &TransparentCache{
		actualPriceService: actualPriceService,
		maxAge:             maxAge,
		prices:             map[string]priceInformation{},
	}
}

// GetPriceFor gets the price for the item, either from the cache or the actual service if it was not cached or too old
func (c *TransparentCache) GetPriceFor(itemCode string) (float64, error) {

	price, ok := c.prices[itemCode]
	if ok {
		if (time.Now().Sub(price.lastReading)) <= c.maxAge {
			return price.value, nil
		}
	}
	v, err := c.actualPriceService.GetPriceFor(itemCode)
	if err != nil {
		return 0, fmt.Errorf("getting price from service : %v", err.Error())
	}

	price = priceInformation{
		lastReading: time.Now(),
		value:       v,
	}

	c.prices[itemCode] = price
	return price.value, nil
}

// GetPricesFor gets the prices for several items at once, some might be found in the cache, others might not
// If any of the operations returns an error, it should return an error as well
func (c *TransparentCache) GetPricesFor(itemCodes ...string) ([]float64, error) {
	results := []float64{}

	ch := make(chan result, len(itemCodes))
	c.getPricesForAsy(ch, itemCodes...)

	for r := range ch {
		if r.err != nil {
			return []float64{}, r.err
		}
		results = append(results, r.price)
	}
	return results, nil
}

func (c *TransparentCache) getPricesForAsy(ch chan result, itemCodes ...string) {

	wg := sync.WaitGroup{}
	wg.Add(len(itemCodes))
	for _, itemCode := range itemCodes {
		go c.getPriceForAsy(itemCode, ch, &wg)
	}
	wg.Wait()
	close(ch)
}

func (c *TransparentCache) getPriceForAsy(itemCode string, ch chan result, wg *sync.WaitGroup) {

	defer wg.Done()

	price, err := c.GetPriceFor(itemCode)
	ch <- result{
		price: price,
		err:   err,
	}

}
