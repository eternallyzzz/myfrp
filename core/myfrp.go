package core

import (
	"context"
	"endpoint/pkg/common"
	"endpoint/pkg/inf"
	"endpoint/pkg/model"
	"endpoint/pkg/zlog"
	"errors"
	"sync"
)

func New(iConfig *model.Config) (*Instance, error) {
	if iConfig.Endpoint == nil && iConfig.Proxy == nil {
		return nil, errors.New("invalid config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	instance := &Instance{Ctx: ctx, Cancel: cancel}

	err := initInstance(instance, iConfig)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func initInstance(ins *Instance, iConfig *model.Config) error {
	cfgs := resolveConfig(iConfig)

	for _, cfg := range cfgs {
		o, err := common.GetServerInstance(ins.Ctx, cfg)
		if err != nil {
			return err
		}

		if future, ok := o.(inf.Future); ok {
			if err := ins.AddTask(future); err != nil {
				return err
			}
		}
	}
	return nil
}

type Instance struct {
	Lock    sync.Mutex
	Ctx     context.Context
	Cancel  context.CancelFunc
	Futures []inf.Future
	Running bool
}

func (i *Instance) AddTask(o inf.Future) error {
	i.Futures = append(i.Futures, o)
	if i.Running {
		if err := o.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (i *Instance) AddTasks(o []inf.Future) error {
	i.Futures = append(i.Futures, o...)
	if i.Running {
		for _, future := range o {
			if err := future.Run(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *Instance) Start() error {
	i.Lock.Lock()
	defer i.Lock.Unlock()

	i.Running = true

	for _, task := range i.Futures {
		if err := task.Run(); err != nil {
			return err
		}
	}

	zlog.Warn("Endpoint started")

	return nil
}

func (i *Instance) Close() error {
	i.Lock.Lock()
	defer i.Lock.Unlock()

	i.Running = false
	i.Cancel()

	var errMsg string
	for _, task := range i.Futures {
		if err := task.Close(); err != nil {
			errMsg += " " + err.Error()
		}
	}

	if errMsg != "" {
		return errors.New(errMsg)
	}

	return nil
}

func resolveConfig(cfg *model.Config) []any {
	cfgs := make([]any, 0)

	if cfg.Endpoint != nil {
		cfgs = append(cfgs, cfg.Endpoint)
	}
	if cfg.Proxy != nil {
		cfgs = append(cfgs, cfg.Proxy)
	}
	return cfgs
}
