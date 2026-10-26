package models

import (
	"errors"
	"strings"

	"github.com/gofrs/uuid"
)

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) createTasks(titles ...string) []*Task {
	tasks := []*Task{}
	for _, title := range titles {
		task := &Task{Title: title}
		verrs, err := ms.DB.ValidateAndCreate(task)
		ms.NoError(err)
		ms.False(verrs.HasAny())
		tasks = append(tasks, task)
	}
	return tasks
}

func (ms *ModelSuite) orderedTitles() []string {
	tasks := Tasks{}
	ms.NoError(ms.DB.Order(TaskOrder).All(&tasks))
	titles := []string{}
	for i, t := range tasks {
		titles = append(titles, t.Title)
		ms.Equal(i+1, t.Position)
	}
	return titles
}

func (ms *ModelSuite) Test_Task_Create_AssignsNextPosition() {
	tasks := ms.createTasks("A", "B", "C")
	for i, task := range tasks {
		ms.Equal(i+1, task.Position)
	}
}

func (ms *ModelSuite) Test_MoveTask_LastToFirst() {
	tasks := ms.createTasks("A", "B", "C")
	ms.NoError(MoveTask(ms.DB, tasks[2].ID, 1))
	ms.Equal([]string{"C", "A", "B"}, ms.orderedTitles())
}

func (ms *ModelSuite) Test_MoveTask_ClampsBeyondEnd() {
	tasks := ms.createTasks("A", "B", "C")
	ms.NoError(MoveTask(ms.DB, tasks[0].ID, 99))
	ms.Equal([]string{"B", "C", "A"}, ms.orderedTitles())
}

func (ms *ModelSuite) Test_MoveTask_SamePosition_NoChange() {
	tasks := ms.createTasks("A", "B", "C")
	ms.NoError(MoveTask(ms.DB, tasks[1].ID, 2))
	ms.Equal([]string{"A", "B", "C"}, ms.orderedTitles())
}

func (ms *ModelSuite) Test_MoveTask_NotFound() {
	ms.createTasks("A")
	err := MoveTask(ms.DB, uuid.Must(uuid.NewV4()), 1)
	ms.True(errors.Is(err, ErrTaskNotFound))
}

func (ms *ModelSuite) Test_MoveTask_DoesNotTouchUpdatedAt() {
	tasks := ms.createTasks("A", "B", "C")
	before := &Task{}
	ms.NoError(ms.DB.Find(before, tasks[2].ID))

	ms.NoError(MoveTask(ms.DB, tasks[2].ID, 1))

	after := &Task{}
	ms.NoError(ms.DB.Find(after, tasks[2].ID))
	ms.True(after.UpdatedAt.Equal(before.UpdatedAt))
	ms.Equal(1, after.Position)
}
