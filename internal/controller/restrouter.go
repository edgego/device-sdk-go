// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2017-2018 Canonical Ltd
// Copyright (C) 2018-2020 IOTech Ltd
// Copyright (c) 2019 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	sdkCommon "github.com/edgexfoundry/device-sdk-go/internal/common"
	"github.com/edgexfoundry/device-sdk-go/internal/controller/correlation"
	v2 "github.com/edgexfoundry/device-sdk-go/internal/v2/controller/http"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/di"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	contractsV2 "github.com/edgexfoundry/go-mod-core-contracts/v2"
	"github.com/gorilla/mux"
)

type RestController struct {
	LoggingClient    logger.LoggingClient
	router           *mux.Router
	reservedRoutes   map[string]bool
	v2HttpController *v2.V2HttpController
}

func NewRestController(r *mux.Router, lc logger.LoggingClient) *RestController {
	return &RestController{
		LoggingClient:    lc,
		router:           r,
		reservedRoutes:   make(map[string]bool),
		v2HttpController: v2.NewV2HttpController(r, lc),
	}
}

func (c *RestController) InitRestRoutes(dic *di.Container) {
	// Status
	c.addReservedRoute(sdkCommon.APIPingRoute, c.statusFunc, dic).Methods(http.MethodGet)
	// Version
	c.addReservedRoute(sdkCommon.APIVersionRoute, c.versionFunc, dic).Methods(http.MethodGet)
	// Command
	c.addReservedRoute(sdkCommon.APIAllCommandRoute, c.commandAllFunc, dic).Methods(http.MethodGet, http.MethodPut)
	c.addReservedRoute(sdkCommon.APIIdCommandRoute, c.commandFunc, dic).Methods(http.MethodGet, http.MethodPut)
	c.addReservedRoute(sdkCommon.APINameCommandRoute, c.commandFunc, dic).Methods(http.MethodGet, http.MethodPut)
	// Callback
	c.addReservedRoute(sdkCommon.APICallbackRoute, c.callbackFunc, dic)
	// Discovery and Transform
	c.addReservedRoute(sdkCommon.APIDiscoveryRoute, c.discoveryFunc, dic).Methods(http.MethodPost)
	c.addReservedRoute(sdkCommon.APITransformRoute, c.transformFunc, dic).Methods(http.MethodGet)
	// Metric and Config
	c.addReservedRoute(sdkCommon.APIMetricsRoute, c.metricsFunc, dic).Methods(http.MethodGet)
	c.addReservedRoute(sdkCommon.APIConfigRoute, c.configFunc, dic).Methods(http.MethodGet)

	c.InitV2RestRoutes(dic)

	c.router.Use(correlation.ManageHeader)
	c.router.Use(correlation.OnResponseComplete)
	c.router.Use(correlation.OnRequestBegin)
}

func (c *RestController) InitV2RestRoutes(dic *di.Container) {
	c.LoggingClient.Info("Registering v2 routes...")

	c.addReservedRoute(contractsV2.ApiPingRoute, c.v2HttpController.Ping, dic).Methods(http.MethodGet)
	c.addReservedRoute(contractsV2.ApiVersionRoute, c.v2HttpController.Version, dic).Methods(http.MethodGet)
	c.addReservedRoute(contractsV2.ApiConfigRoute, c.v2HttpController.Config, dic).Methods(http.MethodGet)
	c.addReservedRoute(contractsV2.ApiMetricsRoute, c.v2HttpController.Metrics, dic).Methods(http.MethodGet)
}

func (c *RestController) addReservedRoute(route string, handler func(http.ResponseWriter, *http.Request, *di.Container), dic *di.Container) *mux.Route {
	c.reservedRoutes[route] = true
	return c.router.HandleFunc(
		route,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), bootstrapContainer.LoggingClientInterfaceName, c.LoggingClient)
			handler(
				w,
				r.WithContext(ctx),
				dic)

		})
}

func (c *RestController) AddRoute(route string, handler func(http.ResponseWriter, *http.Request), methods ...string) error {
	if c.reservedRoutes[route] {
		return errors.New("route is reserved")
	}

	c.router.HandleFunc(
		route,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), bootstrapContainer.LoggingClientInterfaceName, c.LoggingClient)
			handler(
				w,
				r.WithContext(ctx))
		}).Methods(methods...)
	c.LoggingClient.Debug("Route added", "route", route, "methods", fmt.Sprintf("%v", methods))

	return nil
}

func (c *RestController) Router() *mux.Router {
	return c.router
}
