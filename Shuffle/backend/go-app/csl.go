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
const WeekLength = 7

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

type CslChartResponse struct {
	Day   CslExecutionStats `json:"day"`
	Week  CslExecutionStats `json:"week"`
	Month CslExecutionStats `json:"month"`
}

type CslExecutionStats struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failure int64 `json:"failure"`
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
//  1. Does user exist in organization
//
// or
//  2. Does user have support access
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
	return errors.New("user attempting to access an organization they're not a part of")
}

// Handle a request that requires OrgStats, created to reduce code duplication.
// Function returns nil if error occurs and handles error response
//  1. Handle Cors
//  2. Handle Api Authentication
//  3. Retrieves context
//  4. Checks users access to org
//  5. Retrieves and returns org statistics
func handleOrgStatsRequest(resp http.ResponseWriter, request *http.Request) *shuffle.ExecutionInfo {
	if shuffle.HandleCors(resp, request) {
		return nil
	}

	user, err := shuffle.HandleApiAuthentication(resp, request)
	if err != nil {
		log.Printf("[ERROR] Api authentication failed in cslWorkflows: %s", err)
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return nil
	}

	ctx := shuffle.GetContext(request)

	err = checkUserOrgAccess(ctx, user)
	if err != nil {
		resp.WriteHeader(401)
		resp.Write(createCslErrorResponse(err))
		return nil
	}

	orgStats, err := shuffle.GetOrgStatistics(ctx, user.ActiveOrg.Id)
	if err != nil {
		log.Printf("[ERROR] Failed getting stats for org %s: %s", user.ActiveOrg.Id, err)
		resp.WriteHeader(500)
		resp.Write(createCslErrorResponse(err))
		return nil
	}

	return orgStats
}

// Write response status code and JSON response body.
// If error occurs during marshaling handle it and write error response
func marshalAndWriteResponse(response http.ResponseWriter, res interface{}, callingFunctionName string) {
	b, err := json.Marshal(res)
	if err != nil {
		log.Printf("[ERROR] Failed marshaling in %s", callingFunctionName)
		response.WriteHeader(500)
		response.Write(createCslErrorResponse(err))
		return
	}

	response.WriteHeader(200)
	response.Write(b)
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

	marshalAndWriteResponse(resp, res, "cslWorkflows")
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

	marshalAndWriteResponse(resp, res, "cslApps")
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
	orgStats := handleOrgStatsRequest(resp, request)
	if orgStats == nil {
		return
	}

	res := CslResponse{
		Success: true,
		Data: CslApiUsageResponse{
			TotalApiUsage: orgStats.TotalApiUsage,
			DailyApiUsage: orgStats.DailyApiUsage,
		},
	}

	marshalAndWriteResponse(resp, res, "cslApiUsage")
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
	orgStats := handleOrgStatsRequest(resp, request)
	if orgStats == nil {
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

	marshalAndWriteResponse(resp, res, "cslWorkflowExecutions")
}

/*
Dashboard:
Returns day, week and month statistics for workflow total, succesful and failed executions

	{
		"success": true,
		"data": {
			"day": {
				"total": 20,
				"success": 10,
				"failure": 10
			},
			"week": {
			...
			},
			"month": {
			...
			}
		}
	}
*/
func cslWorkflowChart(resp http.ResponseWriter, request *http.Request) {
	orgStats := handleOrgStatsRequest(resp, request)
	if orgStats == nil {
		return
	}

	// calculate the weeks execution stats
	var weekSuccess int64 = orgStats.DailyWorkflowExecutionsFinished
	var weekFailure int64 = orgStats.DailyWorkflowExecutions - orgStats.DailyWorkflowExecutionsFinished

	i := 0
	for i < WeekLength-1 && i < len(orgStats.DailyStatistics) {
		dayStats := orgStats.DailyStatistics[len(orgStats.DailyStatistics)-i-1]
		weekSuccess += dayStats.WorkflowExecutionsFinished
		weekFailure += dayStats.WorkflowExecutions - dayStats.WorkflowExecutionsFinished

		i++
	}

	res := CslResponse{
		Success: true,
		Data: CslChartResponse{
			Day: CslExecutionStats{
				Total:   orgStats.DailyWorkflowExecutions,
				Success: orgStats.DailyWorkflowExecutionsFinished,
				Failure: orgStats.DailyWorkflowExecutions - orgStats.DailyWorkflowExecutionsFinished,
			},
			Week: CslExecutionStats{
				Total:   weekSuccess + weekFailure,
				Success: weekSuccess,
				Failure: weekFailure,
			},
			Month: CslExecutionStats{
				Total:   orgStats.MonthlyWorkflowExecutions,
				Success: orgStats.MonthlyWorkflowExecutionsFinished,
				Failure: orgStats.MonthlyWorkflowExecutions - orgStats.MonthlyWorkflowExecutionsFinished,
			},
		},
	}

	marshalAndWriteResponse(resp, res, "cslWorkflowChart")
}

/*
Dashboard:
Returns day, week and month statistics for app total, succesful and failed executions

	{
		"success": true,
		"data": {
			"day": {
				"total": 30,
				"success": 30,
				"failure": 0
			},
			"week": {
			...
			},
			"month": {
			...
			}
		}
	}
*/
func cslAppChart(resp http.ResponseWriter, request *http.Request) {
	orgStats := handleOrgStatsRequest(resp, request)
	if orgStats == nil {
		return
	}

	// calculate the weeks execution stats
	var weekSuccess int64 = orgStats.DailyAppExecutions - orgStats.DailyAppExecutionsFailed
	var weekFailure int64 = orgStats.DailyAppExecutionsFailed

	i := 0
	for i < WeekLength-1 && i < len(orgStats.DailyStatistics) {
		dayStats := orgStats.DailyStatistics[len(orgStats.DailyStatistics)-i-1]
		weekSuccess += dayStats.AppExecutions - dayStats.AppExecutionsFailed
		weekFailure += dayStats.AppExecutionsFailed

		i++
	}

	res := CslResponse{
		Success: true,
		Data: CslChartResponse{
			Day: CslExecutionStats{
				Total:   orgStats.DailyAppExecutions,
				Success: orgStats.DailyAppExecutions - orgStats.DailyAppExecutionsFailed,
				Failure: orgStats.DailyAppExecutionsFailed,
			},
			Week: CslExecutionStats{
				Total:   weekSuccess + weekFailure,
				Success: weekSuccess,
				Failure: weekFailure,
			},
			Month: CslExecutionStats{
				Total:   orgStats.MonthlyAppExecutions,
				Success: orgStats.MonthlyAppExecutions - orgStats.MonthlyAppExecutionsFailed,
				Failure: orgStats.MonthlyAppExecutionsFailed,
			},
		},
	}

	marshalAndWriteResponse(resp, res, "cslAppChart")
}
