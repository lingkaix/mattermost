// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
)

func (a *App) CreateDesktopToken(token string, createdAt int64) *model.AppError {
	// Check if the token already exists in the database somehow
	// If so return an error
	_, getErr := a.Srv().Store().DesktopTokens().GetUserId(token, 0)
	if getErr == nil {
		return model.NewAppError("CreateDesktopToken", "app.desktop_token.create.collision", nil, "", http.StatusBadRequest)
	}

	// Create token in the database
	err := a.Srv().Store().DesktopTokens().Insert(token, createdAt, nil)
	if err != nil {
		return model.NewAppError("CreateDesktopToken", "app.desktop_token.create.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (a *App) AuthenticateDesktopToken(token string, expiryTime int64, user *model.User) *model.AppError {
	// Throw an error if the token is expired
	err := a.Srv().Store().DesktopTokens().SetUserId(token, expiryTime, user.Id)
	if err != nil {
		// Delete the token if it is expired
		a.Srv().Go(func() {
			a.Srv().Store().DesktopTokens().Delete(token)
		})

		return model.NewAppError("AuthenticateDesktopToken", "app.desktop_token.authenticate.invalid_or_expired", nil, err.Error(), http.StatusBadRequest)
	}

	return nil
}

func (a *App) ValidateDesktopToken(token string, expiryTime int64) (*model.User, *model.AppError) {
	// Check if token is expired
	userId, err := a.Srv().Store().DesktopTokens().GetUserId(token, expiryTime)
	if err != nil {
		// Delete the token if it is expired
		a.Srv().Go(func() {
			a.Srv().Store().DesktopTokens().Delete(token)
		})

		return nil, model.NewAppError("ValidateDesktopToken", "app.desktop_token.validate.expired", nil, err.Error(), http.StatusUnauthorized)
	}

	// If there's no user id, it's not authenticated yet
	if userId == "" {
		return nil, model.NewAppError("ValidateDesktopToken", "app.desktop_token.validate.invalid", nil, "", http.StatusUnauthorized)
	}

	// Get the user profile
	user, userErr := a.GetUser(userId)
	if userErr != nil {
		// Delete the token if the user is invalid somehow
		a.Srv().Go(func() {
			a.Srv().Store().DesktopTokens().Delete(token)
		})

		return nil, model.NewAppError("ValidateDesktopToken", "app.desktop_token.validate.no_user", nil, userErr.Error(), http.StatusInternalServerError)
	}

	// Clean up other tokens if they exist
	a.Srv().Go(func() {
		a.Srv().Store().DesktopTokens().DeleteByUserId(userId)
	})

	return user, nil
}
