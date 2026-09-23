package callback

import (
	"net/http"
)

func (s *Server) handleGetSSToken(w http.ResponseWriter, r *http.Request) {
	var req GetSSTokenReq
	if !s.readJSON(w, r, &req, s.syncFailResp()) {
		return
	}
	if s.cbs.GetSSToken == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.GetSSToken(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleUpdateSSToken(w http.ResponseWriter, r *http.Request) {
	var req UpdateSSTokenReq
	if !s.readJSON(w, r, &req, s.syncFailResp()) {
		return
	}
	if s.cbs.UpdateSSToken == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.UpdateSSToken(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleGetUserInfo(w http.ResponseWriter, r *http.Request) {
	var req GetUserInfoReq
	if !s.readJSON(w, r, &req, s.syncFailResp()) {
		return
	}
	if s.cbs.GetUserInfo == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.GetUserInfo(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleReportGameInfo(w http.ResponseWriter, r *http.Request) {
	var req ReportGameInfoReq
	if !s.readJSON(w, r, &req, s.syncFailResp()) {
		return
	}
	if s.cbs.ReportGameInfo == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	err := s.cbs.ReportGameInfo(r.Context(), req)
	s.writeResult(w, nil, err)
}

func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	var req GetAccountReq
	if !s.readJSON(w, r, &req, s.syncFailResp()) {
		return
	}
	if s.cbs.GetAccount == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.GetAccount(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleGetScore(w http.ResponseWriter, r *http.Request) {
	var req GetScoreReq
	if !s.readJSON(w, r, &req, s.syncFailResp()) {
		return
	}
	if s.cbs.GetScore == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.GetScore(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleUpdateScore(w http.ResponseWriter, r *http.Request) {
	var req UpdateScoreReq
	if !s.readJSON(w, r, &req, s.syncFailResp()) {
		return
	}
	if s.cbs.UpdateScore == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.UpdateScore(r.Context(), req)
	s.writeResult(w, respData, err)
}

// handleNotify 异步通知入口，分发逻辑见 notify.go。
func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	s.dispatchNotify(w, r)
}
