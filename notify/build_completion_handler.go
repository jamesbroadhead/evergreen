package notify

import (
	"github.com/evergreen-ci/evergreen"
	"github.com/evergreen-ci/evergreen/model/build"
	"github.com/evergreen-ci/evergreen/web"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
)

// Handler for build completion notifications, i.e. send notifications whenever
// a build finishes. Implements NotificationHandler from
// notification_handler.go.
type BuildCompletionHandler struct {
	BuildNotificationHandler
	Name string
}

func (self *BuildCompletionHandler) GetNotifications(ae *web.App, key *NotificationKey) ([]Email, error) {
	var emails []Email
	preface := mciCompletionPreface
	if evergreen.IsPatchRequester(key.NotificationRequester) {
		preface = patchCompletionPreface
	}
	triggeredNotifications, err :=
		self.getRecentlyFinishedBuildsWithStatus(key, "", preface, completionSubject)
	if err != nil {
		return nil, err
	}

	for _, triggered := range triggeredNotifications {
		email, err := self.TemplateNotification(ae, &triggered)
		if err != nil {
			grip.Warning(message.WrapError(err, message.Fields{
				"message":      "template error",
				"id":           triggered.Current.Id,
				"notification": self.Name,
				"runner":       RunnerName,
			}))
			continue
		}

		emails = append(emails, email)
	}

	return emails, nil
}

func (self *BuildCompletionHandler) TemplateNotification(ae *web.App, notification *TriggeredBuildNotification) (Email, error) {
	changeInfo, err := self.GetChangeInfo(notification)
	if err != nil {
		return nil, err
	}
	return self.templateNotification(ae, notification, changeInfo)
}

func (self *BuildCompletionHandler) GetChangeInfo(
	notification *TriggeredBuildNotification) ([]ChangeInfo, error) {
	return self.constructChangeInfo([]build.Build{*notification.Current}, &notification.Key)
}
