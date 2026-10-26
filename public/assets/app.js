// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (form.matches("[data-task-move]")) {
    handleMoveSubmit(form, event);
    return;
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var completed = task.classList.toggle("task-completed");
  var button = form.querySelector("button");
  var status = task.querySelector(".task-status");

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }

  fetch(form.action);
});

// Reordenación de tareas: botones Subir/Bajar y arrastre con HTML5 drag and drop.
function moveTaskTo(li, position) {
  var tokenInput = document.querySelector('input[name="authenticity_token"]');
  var body = new URLSearchParams({
    authenticity_token: tokenInput ? tokenInput.value : "",
    _method: "PUT",
    position: String(position),
  });

  fetch(li.dataset.moveUrl, { method: "POST", body: body, credentials: "same-origin" })
    .then(function (response) {
      if (!response.ok) {
        window.location.reload();
      }
    })
    .catch(function () {
      window.location.reload();
    });
}

function refreshMoveControls() {
  var items = document.querySelectorAll(".task-list > .task");
  Array.prototype.forEach.call(items, function (item, index) {
    var up = item.querySelector('[data-task-move="up"]');
    var down = item.querySelector('[data-task-move="down"]');
    if (up) {
      up.querySelector("button").disabled = index === 0;
      up.querySelector('input[name="position"]').value = String(index);
    }
    if (down) {
      down.querySelector("button").disabled = index === items.length - 1;
      down.querySelector('input[name="position"]').value = String(index + 2);
    }
  });
}

function taskIndex(li) {
  return Array.prototype.indexOf.call(li.parentNode.children, li);
}

function announce(li) {
  var region = document.getElementById("reorder-status");
  if (!region) {
    return;
  }
  var title = li.querySelector(".task-title").textContent;
  region.textContent = "\"" + title + "\" movida a la posición " + (taskIndex(li) + 1) + " de " + li.parentNode.children.length + ".";
}

var draggedTask = null;
var draggedFrom = -1;

document.addEventListener("mousedown", function (event) {
  var handle = event.target.closest && event.target.closest("[data-drag-handle]");
  if (handle) {
    handle.closest(".task").draggable = true;
  }
});

document.addEventListener("mouseup", function () {
  var items = document.querySelectorAll(".task[draggable=true]");
  Array.prototype.forEach.call(items, function (item) {
    item.draggable = false;
  });
});

document.addEventListener("dragstart", function (event) {
  var li = event.target.closest && event.target.closest(".task");
  if (!li || !li.draggable) {
    return;
  }
  draggedTask = li;
  draggedFrom = taskIndex(li);
  event.dataTransfer.effectAllowed = "move";
  event.dataTransfer.setData("text/plain", li.dataset.taskId);
  li.classList.add("task-dragging");
});

document.addEventListener("dragover", function (event) {
  if (!draggedTask || !event.target.closest || !event.target.closest(".task-list")) {
    return;
  }
  event.preventDefault();

  var over = event.target.closest(".task");
  if (!over || over === draggedTask) {
    return;
  }
  var rect = over.getBoundingClientRect();
  var after = event.clientY > rect.top + rect.height / 2;
  over.parentNode.insertBefore(draggedTask, after ? over.nextSibling : over);
});

document.addEventListener("drop", function (event) {
  if (draggedTask) {
    event.preventDefault();
  }
});

document.addEventListener("dragend", function () {
  if (!draggedTask) {
    return;
  }
  var li = draggedTask;
  var from = draggedFrom;
  draggedTask = null;
  draggedFrom = -1;
  li.classList.remove("task-dragging");
  li.draggable = false;

  var to = taskIndex(li);
  if (to !== from) {
    refreshMoveControls();
    announce(li);
    moveTaskTo(li, to + 1);
  }
});

function handleMoveSubmit(form, event) {
  event.preventDefault();

  var li = form.closest(".task");
  var direction = form.dataset.taskMove;
  var from = taskIndex(li);
  var to = direction === "up" ? from - 1 : from + 1;
  var list = li.parentNode;
  if (to < 0 || to >= list.children.length) {
    return;
  }

  if (direction === "up") {
    list.insertBefore(li, list.children[to]);
  } else {
    list.insertBefore(li, list.children[to].nextSibling);
  }

  refreshMoveControls();

  var button = form.querySelector("button");
  if (button.disabled) {
    var other = li.querySelector('[data-task-move="' + (direction === "up" ? "down" : "up") + '"] button');
    button = other;
  }
  button.focus();

  announce(li);
  moveTaskTo(li, to + 1);
}
