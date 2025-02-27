/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package restHandler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/pkg/pipeline"
	"github.com/devtron-labs/devtron/pkg/pipeline/types"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

type MigrateDbRestHandler interface {
	SaveDbConfig(w http.ResponseWriter, r *http.Request)
	FetchAllDbConfig(w http.ResponseWriter, r *http.Request)
	FetchOneDbConfig(w http.ResponseWriter, r *http.Request)
	UpdateDbConfig(w http.ResponseWriter, r *http.Request)
	FetchDbConfigForAutoComp(w http.ResponseWriter, r *http.Request)
}
type MigrateDbRestHandlerImpl struct {
	dockerRegistryConfig pipeline.DockerRegistryConfig
	logger               *zap.SugaredLogger
	gitRegistryConfig    pipeline.GitRegistryConfig
	dbConfigService pipeline.DbConfigService
	userAuthService user.UserService
	validator       *validator.Validate
	dbMigrationService   pipeline.DbMigrationService
	enforcer             casbin.Enforcer
}

func NewMigrateDbRestHandlerImpl(dockerRegistryConfig pipeline.DockerRegistryConfig,
	logger *zap.SugaredLogger, gitRegistryConfig pipeline.GitRegistryConfig,
	dbConfigService pipeline.DbConfigService, userAuthService user.UserService,
	validator *validator.Validate, dbMigrationService pipeline.DbMigrationService,
	enforcer casbin.Enforcer) *MigrateDbRestHandlerImpl {
	return &MigrateDbRestHandlerImpl{
		dockerRegistryConfig: dockerRegistryConfig,
		logger:               logger,
		gitRegistryConfig:    gitRegistryConfig,
		dbConfigService:      dbConfigService,
		userAuthService:      userAuthService,
		validator:            validator,
		dbMigrationService:   dbMigrationService,
		enforcer:             enforcer,
	}
}

func (impl MigrateDbRestHandlerImpl) SaveDbConfig(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var bean types.DbConfigBean
	err = decoder.Decode(&bean)
	if err != nil {
		impl.logger.Errorw("request err, SaveDbConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	bean.UserId = userId
	impl.logger.Errorw("request payload, SaveDbConfig", "err", err, "payload", bean)
	err = impl.validator.Struct(bean)
	if err != nil {
		impl.logger.Errorw("validation err, SaveDbConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	// RBAC enforcer applying
	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceMigrate, casbin.ActionCreate, bean.Name); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC enforcer Ends

	res, err := impl.dbConfigService.Save(&bean)
	if err != nil {
		impl.logger.Errorw("service err, SaveDbConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl MigrateDbRestHandlerImpl) FetchAllDbConfig(w http.ResponseWriter, r *http.Request) {
	res, err := impl.dbConfigService.GetAll()
	if err != nil {
		impl.logger.Errorw("service err, FetchAllDbConfig", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC enforcer applying
	token := r.Header.Get("token")
	var result []types.DbConfigBean
	for _, item := range res {
		if ok := impl.enforcer.Enforce(token, casbin.ResourceMigrate, casbin.ActionGet, item.Name); ok {
			result = append(result, *item)
		}
	}
	//RBAC enforcer Ends

	common.WriteJsonResp(w, err, result, http.StatusOK)
}

func (impl MigrateDbRestHandlerImpl) FetchOneDbConfig(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		impl.logger.Errorw("request err, FetchOneDbConfig", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	res, err := impl.dbConfigService.GetById(id)
	if err != nil {
		impl.logger.Errorw("service err, FetchOneDbConfig", "err", err, "id", id)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC enforcer applying
	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceMigrate, casbin.ActionGet, res.Name); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC enforcer Ends

	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl MigrateDbRestHandlerImpl) UpdateDbConfig(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := impl.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var bean types.DbConfigBean

	err = decoder.Decode(&bean)
	if err != nil {
		impl.logger.Errorw("request err, UpdateDbConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	bean.UserId = userId
	impl.logger.Errorw("request payload, UpdateDbConfig", "err", err, "payload", bean)
	err = impl.validator.Struct(bean)
	if err != nil {
		impl.logger.Errorw("service err, UpdateDbConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	// RBAC enforcer applying
	token := r.Header.Get("token")
	if ok := impl.enforcer.Enforce(token, casbin.ResourceMigrate, casbin.ActionUpdate, bean.Name); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC enforcer Ends

	res, err := impl.dbConfigService.Update(&bean)
	if err != nil {
		impl.logger.Errorw("service err, UpdateDbConfig", "err", err, "payload", bean)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (impl MigrateDbRestHandlerImpl) FetchDbConfigForAutoComp(w http.ResponseWriter, r *http.Request) {
	res, err := impl.dbConfigService.GetForAutocomplete()
	if err != nil {
		impl.logger.Errorw("service err, FetchDbConfigForAutoComp", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC enforcer applying
	token := r.Header.Get("token")
	var result []types.DbConfigBean
	for _, item := range res {
		if ok := impl.enforcer.Enforce(token, casbin.ResourceMigrate, casbin.ActionGet, item.Name); ok {
			result = append(result, *item)
		}
	}
	//RBAC enforcer Ends

	common.WriteJsonResp(w, err, result, http.StatusOK)
}
