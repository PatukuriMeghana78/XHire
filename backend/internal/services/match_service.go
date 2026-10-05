package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"job_application_analysis/internal/repositories"
)

type MatchService struct {
	jobs     *repositories.JobRepository
	profiles *repositories.ProfileRepository
	apps     *repositories.ApplicationRepository
}

func NewMatchService(
	jobs *repositories.JobRepository,
	profiles *repositories.ProfileRepository,
	apps *repositories.ApplicationRepository,
) *MatchService {
	return &MatchService{jobs: jobs, profiles: profiles, apps: apps}
}

type skillMatchResult struct {
	MatchScore     int      `json:"matchScore"`
	MatchedSkills  []string `json:"matchedSkills"`
	MissingSkills  []string `json:"missingSkills"`
	ExtraSkills    []string `json:"extraSkills"`
}

func overlapMatch(required, candidate []string) skillMatchResult {
	req := map[string]string{}
	for _, s := range required {
		key := strings.ToLower(strings.TrimSpace(s))
		if key != "" {
			req[key] = s
		}
	}
	cand := map[string]string{}
	for _, s := range candidate {
		key := strings.ToLower(strings.TrimSpace(s))
		if key != "" {
			cand[key] = s
		}
	}
	var matched, missing, extra []string
	for k, orig := range req {
		if _, ok := cand[k]; ok {
			matched = append(matched, orig)
		} else {
			missing = append(missing, orig)
		}
	}
	for k, orig := range cand {
		if _, ok := req[k]; !ok {
			extra = append(extra, orig)
		}
	}
	score := 0
	if len(req) > 0 {
		score = int(float64(len(matched)) / float64(len(req)) * 100)
	}
	if matched == nil {
		matched = []string{}
	}
	if missing == nil {
		missing = []string{}
	}
	return skillMatchResult{MatchScore: score, MatchedSkills: matched, MissingSkills: missing, ExtraSkills: extra}
}

func (s *MatchService) Score(required, candidate []string) skillMatchResult {
	local := overlapMatch(required, candidate)
	payload, _ := json.Marshal(map[string]interface{}{
		"required_skills":  required,
		"candidate_skills": candidate,
	})
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(nlpServiceURL()+"/skill-match", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return local
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return local
	}
	var remote skillMatchResult
	if err := json.NewDecoder(resp.Body).Decode(&remote); err != nil {
		return local
	}
	if remote.MatchedSkills == nil {
		remote.MatchedSkills = local.MatchedSkills
	}
	if remote.MissingSkills == nil {
		remote.MissingSkills = local.MissingSkills
	}
	return remote
}

func (s *MatchService) JobMatch(candidateID, jobID uint) (map[string]interface{}, error) {
	job, err := s.jobs.GetByID(jobID)
	if err != nil {
		return nil, err
	}
	profile, _ := s.profiles.GetCandidateByUserID(candidateID)
	skills := []string{}
	if profile != nil {
		skills = profile.Skills
	}
	result := s.Score(job.RequiredSkills, skills)
	summary := "Skill overlap against the posted requirements."
	return map[string]interface{}{
		"jobId":         strconv.FormatUint(uint64(job.ID), 10),
		"candidateId":   strconv.FormatUint(uint64(candidateID), 10),
		"matchScore":    result.MatchScore,
		"summary":       summary,
		"matchedSkills": result.MatchedSkills,
		"missingSkills": result.MissingSkills,
		"factors": []map[string]interface{}{
			{"label": "Required skills", "weight": 1, "contribution": result.MatchScore, "detail": "Overlap of candidate skills with job requirements"},
		},
	}, nil
}

func (s *MatchService) SkillGaps(candidateID uint, jobID uint) ([]map[string]interface{}, error) {
	profile, err := s.profiles.GetCandidateByUserID(candidateID)
	skills := []string{}
	if err == nil {
		skills = profile.Skills
	}
	var required []string
	if jobID != 0 {
		if job, jerr := s.jobs.GetByID(jobID); jerr == nil {
			required = job.RequiredSkills
		}
	}
	if len(required) == 0 {
		required = []string{"communication", "problem solving"}
	}
	result := s.Score(required, skills)
	out := make([]map[string]interface{}, 0, len(result.MissingSkills)+len(result.MatchedSkills))
	for _, skill := range result.MissingSkills {
		out = append(out, map[string]interface{}{
			"skill": skill, "required": 80, "current": 20, "priority": "high",
		})
	}
	for _, skill := range result.MatchedSkills {
		out = append(out, map[string]interface{}{
			"skill": skill, "required": 80, "current": 75, "priority": "low",
		})
	}
	return out, nil
}

func (s *MatchService) RankJob(jobID uint) ([]map[string]interface{}, error) {
	job, err := s.jobs.GetByID(jobID)
	if err != nil {
		return nil, err
	}
	apps, err := s.apps.FindByJobID(jobID)
	if err != nil {
		return nil, err
	}
	ranked := make([]map[string]interface{}, 0, len(apps))
	for _, app := range apps {
		cid := app.CandidateID
		if cid == 0 {
			cid = app.UserID
		}
		profile, _ := s.profiles.GetCandidateByUserID(cid)
		skills := []string{}
		name := ""
		headline := ""
		if profile != nil {
			skills = profile.Skills
			name = profile.FullName
			headline = profile.Headline
		}
		result := s.Score(job.RequiredSkills, skills)
		_ = s.apps.UpdateMatchScore(app.ID, result.MatchScore)
		reasons := result.MatchedSkills
		if len(reasons) > 3 {
			reasons = reasons[:3]
		}
		ranked = append(ranked, map[string]interface{}{
			"candidateId":    strconv.FormatUint(uint64(cid), 10),
			"candidateName":  name,
			"headline":       headline,
			"matchScore":     result.MatchScore,
			"applicationId":  strconv.FormatUint(uint64(app.ID), 10),
			"topReasons":     reasons,
			"matchedSkills":  result.MatchedSkills,
			"missingSkills":  result.MissingSkills,
		})
	}
	return ranked, nil
}
