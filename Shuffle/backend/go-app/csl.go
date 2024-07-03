package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/shuffle/shuffle-shared"
)

const MaxAppCount = 1000
const MonthLength = 30

type CslResponse struct {
	Success bool        `json:"success"`
	Reason  string      `json:"reason,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type CslWorkflowsResponse struct {
	Workflows           int `json:"workflows"`
	UnexecutedWorkflows int `json:"unexecuted_workflows"`
}

type CslAppsResponse struct {
	Apps int `json:"apps"`
}

type CslApiUsageResponse struct {
	TotalApiUsage int64 `json:"total_api_usage"`
	DailyApiUsage int64 `json:"daily_api_usage"`
}

type CslWorkflowExecutionsResponse struct {
	WorkflowExecutions         int64   `json:"workflow_executions"`
	WorkflowExecutionsFinished int64   `json:"workflow_executions_finished"`
	WorkflowExecutionsFailed   int64   `json:"workflow_executions_failed"`
	DailyWorkflowExecutions    []int64 `json:"daily_workflow_executions"`
}

// Take error and generate response in Csl expected format
func createCslErrorResponse(err error) []byte {
	res := CslResponse{
		Success: false,
		Reason:  err.Error(),
	}

	b, err := json.Marshal(res)
	if err != nil {
		return []byte(`{"success": false, "reason": "Failed marshalling"}`)
	}

	return b
}

// Verifies whether user has access to organization
// 1) Does user exist in organization
// or
// 2) Does user have support access
func checkUserOrgAccess(ctx context.Context, user shuffle.User) error {

	org, err := shuffle.GetOrg(ctx, user.ActiveOrg.Id)
	if err != nil {
		log.Printf("[ERROR] Failed retrieving Org %s: %s", user.ActiveOrg.Id, err)
		return err
	}

	for _, orgUser := range org.Users {
		if orgUser.Id == user.Id {
			return nil
		}
	}

	if user.SupportAccess {
		log.Printf("[AUDIT] User %s (%s) is accessing org %s (%s) with support access", user.Username, user.Id, org.Name, org.Id)
		return nil
	}

	log.Printf("[WARNING] User %s isn't a part of org %s", user.Id, org.Id)
	return errors.New("User attempting to access an organization they're not a part of")
}

// ===========================
//          CSL APIS
// ===========================

// TESTING:
// Test endpoint that returns example success Csl Response, missing auth checks
func cslTestSuccess(resp http.ResponseWriter, request *http.Request) {
	if shuffle.HandleCors(resp, request) {
		return
	}

	res := CslResponse{
		Success: true,
		Data:    "this field can be any type",
	}

	b, err := json.Marshal(res)
	if err != nil {
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	resp.WriteHeader(200)
	resp.Write(b)
}

// TESTING:
// Test endpoint that returns example failure Csl Response, missing auth checks
func cslTestFailure(resp http.ResponseWriter, request *http.Request) {
	if shuffle.HandleCors(resp, request) {
		return
	}

	res := CslResponse{
		Success: false,
		Reason:  "failed because something happened",
	}

	b, err := json.Marshal(res)
	if err != nil {
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	resp.WriteHeader(200)
	resp.Write(b)
}

/*
Dashboard:
Returns workflows belonging to current organization and number of those
workflows that haven't been executed before

	{
	    "success": true,
	    "data": {
	        "workflows": 2,
	        "unexecuted_workflows": 0
	    }
	}
*/
func cslWorkflows(resp http.ResponseWriter, request *http.Request) {
	if shuffle.HandleCors(resp, request) {
		return
	}

	user, err := shuffle.HandleApiAuthentication(resp, request)
	if err != nil {
		log.Printf("[ERROR] Api authentication failed in cslWorkflows: %s", err)
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return
	}

	ctx := shuffle.GetContext(request)

	err = checkUserOrgAccess(ctx, user)
	if err != nil {
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return
	}

	workflows, err := shuffle.GetAllWorkflowsByQuery(ctx, user)
	if err != nil {
		log.Printf("[ERROR] Failed getting workflows for user %s: %s", user.Username, err)
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	unexecutedWorkflows := 0
	for _, workflow := range workflows {

		// amount argument can be hardcoded to 1 since we just need to check
		// if there's been 1 or more executions
		workflowExecutions, err := shuffle.GetAllWorkflowExecutions(ctx, workflow.ID, 1)
		if err != nil {
			log.Printf("[ERROR] Failed getting workflow executions for workflow %s: %s", workflow.ID, err)
			resp.WriteHeader(500)
			resp.Write(createCslErrorResponse(err))
			return
		}

		if len(workflowExecutions) == 0 {
			unexecutedWorkflows++
		}
	}

	res := CslResponse{
		Success: true,
		Data: CslWorkflowsResponse{
			Workflows:           len(workflows),
			UnexecutedWorkflows: unexecutedWorkflows,
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] Failed marshaling in cslWorkflows")
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	resp.WriteHeader(200)
	resp.Write(b)
}

/*
Dashboard:
Returns apps that the current organization has access to and the
number of apps that haven't been executed before

	{
	    "success": true,
	    "data": {
	        "apps": 62
	    }
	}
*/
func cslApps(resp http.ResponseWriter, request *http.Request) {
	if shuffle.HandleCors(resp, request) {
		return
	}

	user, err := shuffle.HandleApiAuthentication(resp, request)
	if err != nil {
		log.Printf("[ERROR] Api authentication failed in cslApps: %s", err)
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return
	}

	ctx := shuffle.GetContext(request)

	err = checkUserOrgAccess(ctx, user)
	if err != nil {
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return
	}

	workflowapps, err := shuffle.GetAllWorkflowApps(ctx, MaxAppCount, 0)
	if err != nil {
		log.Printf("[ERROR] Failed getting all apps: %s", err)
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	// TODO: get the count of apps that haven't been executed before and update comment for function

	res := CslResponse{
		Success: true,
		Data: CslAppsResponse{
			Apps: len(workflowapps),
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] Failed marshaling in cslApps")
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	resp.WriteHeader(200)
	resp.Write(b)
}

/*
Dashboard:
Returns total and daily API usage for the current organization

	{
	    "success": true,
	    "data": {
	        "total_api_usage": 680,
	        "daily_api_usage": 670
	    }
	}
*/
func cslApiUsage(resp http.ResponseWriter, request *http.Request) {
	if shuffle.HandleCors(resp, request) {
		return
	}

	user, err := shuffle.HandleApiAuthentication(resp, request)
	if err != nil {
		log.Printf("[ERROR] Api authentication failed in cslWorkflows: %s", err)
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return
	}

	ctx := shuffle.GetContext(request)

	err = checkUserOrgAccess(ctx, user)
	if err != nil {
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return
	}

	orgStats, err := shuffle.GetOrgStatistics(ctx, user.ActiveOrg.Id)
	if err != nil {
		log.Printf("[ERROR] Failed getting stats for org %s: %s", user.ActiveOrg.Id, err)
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	res := CslResponse{
		Success: true,
		Data: CslApiUsageResponse{
			TotalApiUsage: orgStats.TotalApiUsage,
			DailyApiUsage: orgStats.DailyApiUsage,
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] Failed marshaling in cslApiUsage")
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	resp.WriteHeader(200)
	resp.Write(b)
}

/*
Dashboard:
Returns monthly workflow (total, successful, failed) executions and
a list of the daily workflow execution count for the last 30 days ordered from most recent to oldest

	{
	    "success": true,
	    "data": {
	        "workflow_executions": 20,
	        "workflow_executions_finished": 10,
	        "workflow_executions_failed": 10,
	        "daily_workflow_executions": [
	            20,
	            ...
	        ]
	    }
	}
*/
func cslWorkflowExecutions(resp http.ResponseWriter, request *http.Request) {
	if shuffle.HandleCors(resp, request) {
		return
	}

	user, err := shuffle.HandleApiAuthentication(resp, request)
	if err != nil {
		log.Printf("[ERROR] Api authentication failed in cslWorkflows: %s", err)
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return
	}

	ctx := shuffle.GetContext(request)

	err = checkUserOrgAccess(ctx, user)
	if err != nil {
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return
	}

	orgStats, err := shuffle.GetOrgStatistics(ctx, user.ActiveOrg.Id)
	if err != nil {
		log.Printf("[ERROR] Failed getting stats for org %s: %s", user.ActiveOrg.Id, err)
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	// add current days value since it's not saved in orgStats.DailyStatistics
	// iterate backwards through list since most recent date is at end of []orgStats.DailyStatistics
	var dailyWorkflowExecutions []int64
	dailyWorkflowExecutions = append(dailyWorkflowExecutions, orgStats.DailyWorkflowExecutions)

	i := 0
	for i < len(orgStats.DailyStatistics) && i < MonthLength {
		dailyWorkflowExecutions = append(dailyWorkflowExecutions, orgStats.DailyStatistics[len(orgStats.DailyStatistics)-i-1].WorkflowExecutions)
		i++
	}

	res := CslResponse{
		Success: true,
		Data: CslWorkflowExecutionsResponse{
			WorkflowExecutions:         orgStats.MonthlyWorkflowExecutions,
			WorkflowExecutionsFinished: orgStats.MonthlyWorkflowExecutionsFinished,
			WorkflowExecutionsFailed:   orgStats.MonthlyWorkflowExecutions - orgStats.MonthlyWorkflowExecutionsFinished,
			DailyWorkflowExecutions:    dailyWorkflowExecutions,
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] Failed marshaling in cslWorkflowExecutions")
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return
	}

	resp.WriteHeader(200)
	resp.Write(b)
}
