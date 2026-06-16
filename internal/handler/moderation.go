package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
)

type ModerationHandler struct {
	reports       *repository.ReportRepository
	posts         *repository.PostRepository
	comments      *repository.CommentRepository
	users         *repository.UserRepository
	sessions      *repository.SessionRepository
	moderation    *repository.ModerationRepository
	notifications *repository.NotificationRepository
	reviewRepo    *repository.ContentReviewRepository
}

func NewModerationHandler(db *sql.DB) *ModerationHandler {
	return &ModerationHandler{
		reports:       repository.NewReportRepository(db),
		posts:         repository.NewPostRepository(db),
		comments:      repository.NewCommentRepository(db),
		users:         repository.NewUserRepository(db),
		sessions:      repository.NewSessionRepository(db),
		moderation:    repository.NewModerationRepository(db),
		notifications: repository.NewNotificationRepository(db),
		reviewRepo:    repository.NewContentReviewRepository(db),
	}
}

// errActionHandled signals that the HTTP response is already written.
var errActionHandled = errors.New("handled")

// --- View data types ---

type ModerationUserRow struct {
	ID       string
	Username string
	Mail     string
	Role     string
	Status   string
}

type ModerationPostRow struct {
	ID       string
	Username string
	Post     string
	Date     string
	Report   string
}

type ModerationReportRow struct {
	ID       string
	Username string
	Reason   string
}

type ModerationPageData struct {
	User            *model.User
	Users           []ModerationUserRow
	Posts           []ModerationPostRow
	Reports         []ModerationReportRow
	PendingPosts    []model.PendingContent
	PendingComments []model.PendingContent
}

// --- Permission guards ---

func (h *ModerationHandler) requireModerator(w http.ResponseWriter, r *http.Request) *model.User {
	user := userFromSession(r, h.sessions, h.users)
	if user == nil || !user.CanModerate() {
		http.Error(w, "Accès interdit", http.StatusForbidden)
		return nil
	}
	return user
}

func (h *ModerationHandler) requireAdmin(w http.ResponseWriter, r *http.Request) *model.User {
	user := userFromSession(r, h.sessions, h.users)
	if user == nil || !user.IsAdmin() {
		http.Error(w, "Accès interdit", http.StatusForbidden)
		return nil
	}
	return user
}

// --- Report handlers ---

func (h *ModerationHandler) ReportPost(w http.ResponseWriter, r *http.Request) {
	h.createReport(w, r, model.TargetPost)
}

func (h *ModerationHandler) ReportComment(w http.ResponseWriter, r *http.Request) {
	h.createReport(w, r, model.TargetComment)
}

func (h *ModerationHandler) ReportUser(w http.ResponseWriter, r *http.Request) {
	h.createReport(w, r, model.TargetUser)
}

func (h *ModerationHandler) createReport(w http.ResponseWriter, r *http.Request, targetType model.TargetType) {
	user := userFromSession(r, h.sessions, h.users)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	targetID := r.PathValue("id")
	category := model.ReportCategory(r.FormValue("category"))
	reason := r.FormValue("reason")

	validCategories := map[model.ReportCategory]bool{
		model.CategorySpam:                 true,
		model.CategoryHarassment:           true,
		model.CategoryInappropriateContent: true,
		model.CategoryHateSpeech:           true,
		model.CategoryOther:                true,
	}
	if !validCategories[category] {
		http.Error(w, "Catégorie invalide", http.StatusBadRequest)
		return
	}

	already, err := h.reports.AlreadyReported(user.ID, targetType, targetID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	if already {
		http.Error(w, "Vous avez déjà signalé cet élément", http.StatusConflict)
		return
	}

	report := &model.Report{
		ID:         utils.NewUUID(),
		ReporterID: user.ID,
		TargetType: targetType,
		TargetID:   targetID,
		Category:   category,
		Reason:     reason,
		Status:     model.ReportPending,
		CreatedAt:  time.Now(),
	}
	if err := h.reports.Create(report); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	redirectTo := r.FormValue("redirect_to")
	if redirectTo == "" {
		redirectTo = "/"
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

// --- Dashboard ---

func (h *ModerationHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := h.requireModerator(w, r)
	if user == nil {
		return
	}

	pendingPosts, _ := h.reviewRepo.GetPendingPosts("asc")
	pendingComments, _ := h.reviewRepo.GetPendingComments("asc")

	data := ModerationPageData{
		User:            user,
		Users:           h.buildUserRows(),
		Posts:           h.buildPostRows(),
		Reports:         h.buildReportRows(),
		PendingPosts:    pendingPosts,
		PendingComments: pendingComments,
	}

	funcMap := template.FuncMap{
		"notifCount": func(u *model.User) int {
			if u == nil {
				return 0
			}
			count, err := h.notifications.CountUnread(u.ID)
			if err != nil {
				return 0
			}
			return count
		},
	}
	tmpl, err := template.New("base").Funcs(funcMap).ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "moderation", "pagemoderation.html"),
	)
	if err != nil {
		log.Printf("parse moderation dashboard template: %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("execute moderation dashboard template: %v", err)
	}
}

func (h *ModerationHandler) buildUserRows() []ModerationUserRow {
	allUsers, _ := h.moderation.GetAllUsers()
	var rows []ModerationUserRow
	for _, u := range allUsers {
		status, _ := h.moderation.GetUserStatus(u.ID)
		rows = append(rows, ModerationUserRow{
			ID:       u.ID,
			Username: u.Username,
			Mail:     u.Email,
			Role:     string(u.Role),
			Status:   status,
		})
	}
	return rows
}

func (h *ModerationHandler) buildPostRows() []ModerationPostRow {
	reportedPosts, _ := h.moderation.GetReportedPosts()
	var rows []ModerationPostRow
	for _, p := range reportedPosts {
		author, _ := h.users.GetByID(p.UserID)
		username := "Inconnu"
		if author != nil {
			username = author.Username
		}
		count, _ := h.moderation.GetReportCountForPost(p.ID)
		rows = append(rows, ModerationPostRow{
			ID:       p.ID,
			Username: username,
			Post:     p.Content,
			Date:     p.CreatedAt.Format("02/01/2006"),
			Report:   strconv.Itoa(count),
		})
	}
	return rows
}

func (h *ModerationHandler) buildReportRows() []ModerationReportRow {
	pending, _ := h.reports.GetPending()
	var rows []ModerationReportRow
	for _, rep := range pending {
		if rep.TargetType != model.TargetUser {
			continue
		}
		rows = append(rows, ModerationReportRow{
			ID:       rep.ID,
			Username: rep.ReporterName,
			Reason:   rep.Reason,
		})
	}
	return rows
}

// --- ResolveReport ---

func (h *ModerationHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	moderator := h.requireModerator(w, r)
	if moderator == nil {
		return
	}

	reportID := r.PathValue("id")
	report, err := h.reports.GetByID(reportID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	actionType := model.ActionType(r.FormValue("action_type"))
	reason := r.FormValue("reason")

	targetUserID, targetPostID, targetCommentID, err := h.resolveTargetIDs(report)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	if err := h.applyReportAction(w, r, actionType, reason, report, moderator, targetUserID, targetPostID, targetCommentID); err != nil {
		return
	}

	action := &model.ModerationAction{
		ID:          utils.NewUUID(),
		ModeratorID: moderator.ID,
		ReportID:    reportID,
		ActionType:  actionType,
		TargetID:    targetUserID,
		Reason:      reason,
		CreatedAt:   time.Now(),
	}
	if err := h.reports.CreateAction(action); err != nil {
		log.Printf("record moderation action for report %q: %v", reportID, err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	if err := h.reports.UpdateStatus(reportID, model.ReportResolved); err != nil {
		log.Printf("resolve moderation report %q: %v", reportID, err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

// applyReportAction exécute l'action de modération. Retourne errActionHandled si la réponse HTTP
// a déjà été écrite (erreur ou action invalide) — le caller doit alors faire return.
func (h *ModerationHandler) applyReportAction(
	w http.ResponseWriter, r *http.Request,
	actionType model.ActionType, reason string,
	report *model.Report, moderator *model.User,
	targetUserID, targetPostID, targetCommentID string,
) error {
	switch actionType {
	case model.ActionDeletePost:
		if err := h.posts.Delete(report.TargetID); err != nil && err != sql.ErrNoRows {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return errActionHandled
		}
		h.notify(targetUserID, moderator.ID, "moderation_delete_post", "", "")

	case model.ActionDeleteComment:
		if err := h.comments.Delete(report.TargetID); err != nil && err != sql.ErrNoRows {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return errActionHandled
		}
		h.notify(targetUserID, moderator.ID, "moderation_delete_comment", "", "")

	case model.ActionWarnUser:
		h.notify(targetUserID, moderator.ID, "moderation_warn", targetPostID, targetCommentID)

	case model.ActionMuteUser:
		durationHours, _ := strconv.Atoi(r.FormValue("duration_hours"))
		if durationHours <= 0 {
			durationHours = 24
		}
		expiresAt := time.Now().Add(time.Duration(durationHours) * time.Hour)
		restriction := &model.UserRestriction{
			ID:        utils.NewUUID(),
			UserID:    targetUserID,
			Type:      model.RestrictionMute,
			Reason:    reason,
			ExpiresAt: &expiresAt,
			CreatedBy: moderator.ID,
			CreatedAt: time.Now(),
		}
		if err := h.reports.CreateRestriction(restriction); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return errActionHandled
		}
		h.notify(targetUserID, moderator.ID, "moderation_mute", "", "")

	case model.ActionBanUser:
		if !moderator.IsAdmin() {
			http.Error(w, "Seul un admin peut bannir un utilisateur", http.StatusForbidden)
			return errActionHandled
		}
		restriction := &model.UserRestriction{
			ID:        utils.NewUUID(),
			UserID:    targetUserID,
			Type:      model.RestrictionBan,
			Reason:    reason,
			ExpiresAt: nil,
			CreatedBy: moderator.ID,
			CreatedAt: time.Now(),
		}
		if err := h.reports.CreateRestriction(restriction); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return errActionHandled
		}
		h.notify(targetUserID, moderator.ID, "moderation_ban", "", "")

	default:
		http.Error(w, "Action invalide", http.StatusBadRequest)
		return errActionHandled
	}
	return nil
}

func (h *ModerationHandler) DismissReport(w http.ResponseWriter, r *http.Request) {
	moderator := h.requireModerator(w, r)
	if moderator == nil {
		return
	}

	reportID := r.PathValue("id")
	if err := h.reports.UpdateStatus(reportID, model.ReportDismissed); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

// --- Content moderation queue ---

func (h *ModerationHandler) PendingQueue(w http.ResponseWriter, r *http.Request) {
	moderator := h.requireModerator(w, r)
	if moderator == nil {
		return
	}

	sort := r.URL.Query().Get("sort")
	if sort != "desc" {
		sort = "asc"
	}

	posts, err := h.reviewRepo.GetPendingPosts(sort)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	comments, err := h.reviewRepo.GetPendingComments(sort)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSON(w, map[string]any{
		"sort":  sort,
		"items": append(posts, comments...),
	})
}

func (h *ModerationHandler) ApproveContent(w http.ResponseWriter, r *http.Request) {
	moderator := h.requireModerator(w, r)
	if moderator == nil {
		return
	}

	contentID := r.PathValue("id")
	contentType := r.FormValue("content_type")

	var err error
	switch contentType {
	case "post":
		err = h.reviewRepo.ApprovePost(contentID)
	case "comment":
		err = h.reviewRepo.ApproveComment(contentID)
	default:
		http.Error(w, "Type invalide", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	h.reviewRepo.CreateReview(&model.ContentReview{
		ID:          utils.NewUUID(),
		ContentType: contentType,
		ContentID:   contentID,
		ReviewerID:  moderator.ID,
		Decision:    "approved",
		CreatedAt:   time.Now(),
	})

	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

func (h *ModerationHandler) RejectContent(w http.ResponseWriter, r *http.Request) {
	moderator := h.requireModerator(w, r)
	if moderator == nil {
		return
	}

	contentID := r.PathValue("id")
	contentType := r.FormValue("content_type")
	reason := r.FormValue("reason")

	var authorID string
	var err error

	switch contentType {
	case "post":
		authorID, err = h.reviewRepo.GetPostAuthor(contentID)
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		err = h.reviewRepo.RejectPost(contentID)
	case "comment":
		authorID, err = h.reviewRepo.GetCommentAuthor(contentID)
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		err = h.reviewRepo.RejectComment(contentID)
	default:
		http.Error(w, "Type invalide", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	h.reviewRepo.CreateReview(&model.ContentReview{
		ID:          utils.NewUUID(),
		ContentType: contentType,
		ContentID:   contentID,
		ReviewerID:  moderator.ID,
		Decision:    "rejected",
		Reason:      reason,
		CreatedAt:   time.Now(),
	})

	h.notify(authorID, moderator.ID, "moderation_content_rejected", "", "")

	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

// --- User management ---

func (h *ModerationHandler) ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	admin := h.requireAdmin(w, r)
	if admin == nil {
		return
	}

	targetUserID := r.PathValue("id")
	newRole := model.Role(r.FormValue("role"))

	if newRole != model.RoleUser && newRole != model.RoleModerator && newRole != model.RoleAdmin {
		http.Error(w, "Rôle invalide", http.StatusBadRequest)
		return
	}

	if err := h.users.UpdateRole(targetUserID, newRole); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	action := &model.ModerationAction{
		ID:          utils.NewUUID(),
		ModeratorID: admin.ID,
		ActionType:  model.ActionChangeRole,
		TargetID:    targetUserID,
		Reason:      string(newRole),
		CreatedAt:   time.Now(),
	}
	if err := h.reports.CreateAction(action); err != nil {
		log.Printf("record role change for user %q: %v", targetUserID, err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

func (h *ModerationHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	moderator := h.requireModerator(w, r)
	if moderator == nil {
		return
	}
	targetID := r.PathValue("id")
	if err := h.moderation.LiftRestriction(targetID); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

func (h *ModerationHandler) KickUser(w http.ResponseWriter, r *http.Request) {
	moderator := h.requireModerator(w, r)
	if moderator == nil {
		return
	}
	targetID := r.PathValue("id")
	if err := h.sessions.DeleteByUserID(targetID); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

func (h *ModerationHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	admin := h.requireAdmin(w, r)
	if admin == nil {
		return
	}
	targetID := r.PathValue("id")
	reason := r.FormValue("reason")
	if reason == "" {
		reason = "Banni par un administrateur"
	}
	if err := h.moderation.BanUser(targetID, admin.ID, reason); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

func (h *ModerationHandler) MuteUser(w http.ResponseWriter, r *http.Request) {
	moderator := h.requireModerator(w, r)
	if moderator == nil {
		return
	}
	targetID := r.PathValue("id")
	reason := r.FormValue("reason")
	if reason == "" {
		reason = "Muté par un administrateur"
	}
	durationHours, _ := strconv.Atoi(r.FormValue("duration_hours"))
	if durationHours <= 0 {
		durationHours = 24
	}
	if err := h.moderation.MuteUser(targetID, moderator.ID, reason, durationHours); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

// --- Helpers ---

func (h *ModerationHandler) resolveTargetIDs(report *model.Report) (userID, postID, commentID string, err error) {
	switch report.TargetType {
	case model.TargetUser:
		userID = report.TargetID
	case model.TargetPost:
		post, e := h.posts.GetByID(report.TargetID)
		if e != nil {
			return "", "", "", e
		}
		userID = post.UserID
		postID = report.TargetID
	case model.TargetComment:
		comment, e := h.comments.GetByID(report.TargetID)
		if e != nil {
			return "", "", "", e
		}
		userID = comment.UserID
		postID = comment.PostID
		commentID = report.TargetID
	}
	return
}

func (h *ModerationHandler) notify(userID, actorID, notifType, postID, commentID string) {
	if userID == "" || userID == actorID {
		return
	}
	if err := h.notifications.Create(&model.Notification{
		ID:        utils.NewUUID(),
		UserID:    userID,
		ActorID:   actorID,
		Type:      notifType,
		PostID:    postID,
		CommentID: commentID,
		CreatedAt: time.Now(),
	}); err != nil {
		log.Printf("create moderation notification for user %q: %v", userID, err)
	}
}

func encodeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode json: %v", err)
	}
}
