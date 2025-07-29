package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/pirogoeth/apps/orba/types"
	api "github.com/pirogoeth/apps/pkg/apitools"
)

type v1Users struct {
	*types.ApiContext
}

func (e *v1Users) RegisterRoutesTo(router *gin.RouterGroup) {
	router.POST("/user", api.ErrorWrapEndpoint(e.createUser))
}

type createUserParams struct {
	Name string `json:"name"`
}

func (e *v1Users) createUser(ctx *gin.Context) error {
	var params createUserParams
	if err := ctx.ShouldBind(&params); err != nil {
		return fmt.Errorf("%s: %w", api.MsgInvalidParameter, err)
	}

	logrus.Debugf("bound params: %#v", params)

	user, err := e.Querier.CreateUser(ctx.Request.Context(), params.Name)
	if err != nil {
		return fmt.Errorf("could not create user: %w", err)
	}

	api.Ok(ctx, &api.Body{"user": user})
	return nil
}
