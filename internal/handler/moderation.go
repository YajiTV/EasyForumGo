package handler

import (
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
	"fmt"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
)

type ModerationHandler struct {
	reports    *repository.ReportRepository
	posts      *repository.PostRepository
	comments   *repository.CommentRepository
	users      *repository.UserRepository
	sessions   *repository.SessionRepository
	moderation *repository.ModerationRepository
	notifications *repository.NotificationRepository }

func NewModerationHandler(db *sql.DB) *ModerationHandler {
	return &ModerationHandler{
		reports:    repository.NewReportRepository(db),
		posts:      repository.NewPostRepository(db),
		comments:   repository.NewCommentRepository(db),
		users:      repository.NewUserRepository(db),
		sessions:   repository.NewSessionRepository(db),
		moderation: repository.NewModerationRepository(db),
		notifications: repository.NewNotificationRepository(db),
	}
}

// ModerationUserRow is the view data for one user row in the moderation page.
type ModerationUserRow struct {
	ID       string
	Username string
	Mail     string
	Role     string
	Status   string
}

// ModerationPostRow is the view data for one reported post row.
type ModerationPostRow struct {
	ID       string
	Username string
	Post     string
	Date     string
	Report   string
}

// ModerationReportRow is the view data for one pending report row.
type ModerationReportRow struct {
	ID       string
	Username string
	Reason   string
}

// ModerationPageData holds all data passed to pagemoderation.html.
type ModerationPageData struct {
	CurrentUser *model.User
	Users       []ModerationUserRow
	Posts       []ModerationPostRow
	Reports     []ModerationReportRow
}

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
	user := h.userFromSession(r)
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

func (h *ModerationHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil || !user.CanModerate() {
		http.Error(w, "Accès interdit", http.StatusForbidden)
		return
	}

	allUsers, _ := h.moderation.GetAllUsers()
	var userRows []ModerationUserRow
	for _, u := range allUsers {
		status, _ := h.moderation.GetUserStatus(u.ID)
		userRows = append(userRows, ModerationUserRow{
			ID:       u.ID,
			Username: u.Username,
			Mail:     u.Email,
			Role:     string(u.Role),
			Status:   status,
		})
	}

	reportedPosts, _ := h.moderation.GetReportedPosts()
	var postRows []ModerationPostRow
	for _, p := range reportedPosts {
		author, _ := h.users.GetByID(p.UserID)
		username := "Inconnu"
		if author != nil {
			username = author.Username
		}
		count, _ := h.moderation.GetReportCountForPost(p.ID)
		postRows = append(postRows, ModerationPostRow{
			ID:       p.ID,
			Username: username,
			Post:     p.Content,
			Date:     p.CreatedAt.Format("02/01/2006"),
			Report:   strconv.Itoa(count),
		})
	}

	pending, _ := h.reports.GetPending()
	var reportRows []ModerationReportRow
	for _, rep := range pending {
		if rep.TargetType != model.TargetUser {
			continue
		}
		reportRows = append(reportRows, ModerationReportRow{
			ID:       rep.ID,
			Username: rep.ReporterName,
			Reason:   rep.Reason,
		})
	}

	data := ModerationPageData{
		CurrentUser: user,
		Users:       userRows,
		Posts:       postRows,
		Reports:     reportRows,
	}

	tmpl, err := template.New("base").ParseFiles(
    filepath.Join("web", "templates", "layout", "base.html"),
    filepath.Join("web", "templates", "moderation", "pagemoderation.html"),
	)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}

func (h *ModerationHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	moderator := h.userFromSession(r)
	if moderator == nil || !moderator.CanModerate() {
		http.Error(w, "Accès interdit", http.StatusForbidden)
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

	switch actionType {
	case model.ActionDeletePost:
		if err := h.posts.Delete(report.TargetID); err != nil && err != sql.ErrNoRows {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		h.notify(targetUserID, moderator.ID, "moderation_delete_post", "", "")

	case model.ActionDeleteComment:
		if err := h.comments.Delete(report.TargetID); err != nil && err != sql.ErrNoRows {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
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
			return
		}
		h.notify(targetUserID, moderator.ID, "moderation_mute", "", "")

	case model.ActionBanUser:
		if !moderator.IsAdmin() {
			http.Error(w, "Seul un admin peut bannir un utilisateur", http.StatusForbidden)
			return
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
			return
		}
		h.notify(targetUserID, moderator.ID, "moderation_ban", "", "")

	default:
		http.Error(w, "Action invalide", http.StatusBadRequest)
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
	h.reports.CreateAction(action)
	h.reports.UpdateStatus(reportID, model.ReportResolved)

	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

func (h *ModerationHandler) DismissReport(w http.ResponseWriter, r *http.Request) {
	moderator := h.userFromSession(r)
	if moderator == nil || !moderator.CanModerate() {
		http.Error(w, "Accès interdit", http.StatusForbidden)
		return
	}

	reportID := r.PathValue("id")
	if err := h.reports.UpdateStatus(reportID, model.ReportDismissed); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

func (h *ModerationHandler) ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	admin := h.userFromSession(r)
	if admin == nil || !admin.IsAdmin() {
		http.Error(w, "Accès interdit", http.StatusForbidden)
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
	h.reports.CreateAction(action)

	http.Redirect(w, r, "/moderation", http.StatusSeeOther)
}

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
	n := &model.Notification{
		ID:        utils.NewUUID(),
		UserID:    userID,
		ActorID:   actorID,
		Type:      notifType,
		PostID:    postID,
		CommentID: commentID,
		CreatedAt: time.Now(),
	}
	_ = h.notifications.Create(n)
}

func (h *ModerationHandler) userFromSession(r *http.Request) *model.User {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil
	}
	session, err := h.sessions.GetByToken(cookie.Value)
	if err != nil || session.ExpiresAt.Before(time.Now()) {
		return nil
	}
	user, err := h.users.GetByID(session.UserID)
	if err != nil {
		return nil
	}
	return user
}
