package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// TaskOrder es el orden en que se listan las tareas; created_at e id solo desempatan.
const TaskOrder = "position asc, created_at asc, id asc"

// ErrTaskNotFound indica que la tarea a mover no existe.
var ErrTaskNotFound = errors.New("tarea no encontrada")

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Position  int       `json:"position" db:"position" form:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	return verrs, nil
}

// BeforeCreate coloca la tarea nueva al final de la lista.
func (t *Task) BeforeCreate(tx *pop.Connection) error {
	if t.Position != 0 {
		return nil
	}

	var max int
	if err := tx.Store.Get(&max, "SELECT COALESCE(MAX(position), 0) FROM tasks"); err != nil {
		return err
	}
	t.Position = max + 1
	return nil
}

// MoveTask mueve la tarea a la posición indicada (1-based, acotada a la lista)
// y renumera las demás sin tocar updated_at.
func MoveTask(tx *pop.Connection, id uuid.UUID, position int) error {
	tasks := Tasks{}
	if err := tx.RawQuery("SELECT * FROM tasks ORDER BY " + TaskOrder + " FOR UPDATE").All(&tasks); err != nil {
		return err
	}

	from := -1
	for i, t := range tasks {
		if t.ID == id {
			from = i
			break
		}
	}
	if from == -1 {
		return ErrTaskNotFound
	}

	if position < 1 {
		position = 1
	}
	if position > len(tasks) {
		position = len(tasks)
	}

	moved := tasks[from]
	rest := append(Tasks{}, tasks[:from]...)
	rest = append(rest, tasks[from+1:]...)
	ordered := append(Tasks{}, rest[:position-1]...)
	ordered = append(ordered, moved)
	ordered = append(ordered, rest[position-1:]...)

	for i, t := range ordered {
		if t.Position == i+1 {
			continue
		}
		if err := tx.RawQuery("UPDATE tasks SET position = ? WHERE id = ?", i+1, t.ID).Exec(); err != nil {
			return err
		}
	}
	return nil
}
