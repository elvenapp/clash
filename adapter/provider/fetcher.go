//go:build foss

package provider

import (
	"time"

	"clash-foss/component/fs"
	types "clash-foss/constant/provider"
	"clash-foss/log"
)

type parser = func([]byte) (any, error)

type fetcher struct {
	holder       fs.ProviderFS
	name         string
	providerType types.ProviderType
	vehicle      types.Vehicle
	interval     time.Duration
	updatedAt    time.Time
	ticker       *time.Ticker
	done         chan struct{}
	parser       parser
	onUpdate     func(any)
}

func (f *fetcher) Name() string {
	return f.name
}

func (f *fetcher) VehicleType() types.VehicleType {
	return f.vehicle.Type()
}

func (f *fetcher) Initial() (any, error) {
	type DataSource func() ([]byte, time.Time, error)

	parseWith := func(dataSource DataSource) ([]byte, any, time.Time, error) {
		data, updatedAt, err := dataSource()
		if err != nil {
			return nil, nil, time.Time{}, err
		}

		parsed, err := f.parser(data)
		if err != nil {
			return nil, nil, time.Time{}, err
		}

		return data, parsed, updatedAt, nil
	}

	isLocal := true

	data, proxies, updatedAt, err := parseWith(func() ([]byte, time.Time, error) {
		return f.holder.Read(f.providerType, f.name)
	})
	if err != nil {
		data, proxies, updatedAt, err = parseWith(func() ([]byte, time.Time, error) {
			data, err := f.vehicle.Read()
			return data, time.Now(), err
		})

		isLocal = false
	}
	if err != nil {
		return nil, err
	}

	if !isLocal {
		err = f.holder.Write(f.providerType, f.name, data)
		if err != nil {
			return nil, err
		}
	}

	f.updatedAt = time.UnixMilli(updatedAt.UnixMilli())

	// pull proxies automatically
	if f.ticker != nil {
		go f.pullLoop()
	}

	return proxies, nil
}

func (f *fetcher) Update() (any, bool, error) {
	buf, err := f.vehicle.Read()
	if err != nil {
		return nil, false, err
	}

	proxies, err := f.parser(buf)
	if err != nil {
		return nil, false, err
	}

	if f.vehicle.Type() != types.File {
		err = f.holder.Write(f.providerType, f.name, buf)
		if err != nil {
			return nil, false, err
		}
	}

	f.updatedAt = time.UnixMilli(time.Now().UnixMilli())

	return proxies, false, nil
}

func (f *fetcher) Destroy() error {
	if f.ticker != nil {
		f.done <- struct{}{}
	}
	return nil
}

func (f *fetcher) pullLoop() {
	for {
		select {
		case <-f.ticker.C:
			if time.Since(f.updatedAt) > f.interval {
				elm, same, err := f.Update()
				if err != nil {
					log.Warnln("[Provider] %s pull error: %s", f.Name(), err.Error())
					return
				}

				if same {
					log.Debugln("[Provider] %s's proxies doesn't change", f.Name())
					return
				}

				log.Infoln("[Provider] %s's proxies update", f.Name())
				if f.onUpdate != nil {
					f.onUpdate(elm)
				}
			}
		case <-f.done:
			f.ticker.Stop()
			return
		}
	}
}

func newFetcher(name string, interval time.Duration, vehicle types.Vehicle, providerType types.ProviderType, parser parser, onUpdate func(any), holder fs.ProviderFS) *fetcher {
	var ticker *time.Ticker
	if interval != 0 {
		ticker = time.NewTicker(time.Minute * 15)
	}

	return &fetcher{
		holder:       holder,
		name:         name,
		providerType: providerType,
		vehicle:      vehicle,
		interval:     interval,
		ticker:       ticker,
		done:         make(chan struct{}, 1),
		parser:       parser,
		onUpdate:     onUpdate,
	}
}
