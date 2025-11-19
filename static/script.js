document.addEventListener('DOMContentLoaded', () => {
    const uploadForm = document.getElementById('upload-form');
    const imageInput = document.getElementById('image-input');
    const uploadStatus = document.getElementById('upload-status');
    const tasksContainer = document.getElementById('tasks-container');

    // Хранилище для отслеживания интервалов опроса по ID задачи
    const pollingIntervals = {};

    // Загрузка и отображение существующих задач при старте
    loadInitialTasks();

    // Обработчик отправки формы
    uploadForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        uploadStatus.textContent = 'Загрузка...';
        const formData = new FormData();
        formData.append('image', imageInput.files[0]);

        try {
            const response = await fetch('/upload', {
                method: 'POST',
                body: formData,
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Ошибка при загрузке файла.');
            }

            const data = await response.json();
            const { taskID } = data;

            uploadStatus.textContent = `Файл успешно загружен! ID задачи: ${taskID}`;
            imageInput.value = ''; // Сбрасываем поле ввода

            saveTask(taskID);
            renderTask({ id: taskID, status: 'PROCESSING' }); // Оптимистичное отображение
            pollStatus(taskID);

        } catch (error) {
            uploadStatus.textContent = `Ошибка: ${error.message}`;
            console.error('Upload error:', error);
        }
    });

    // Функция опроса статуса задачи
    function pollStatus(taskID) {
        if (pollingIntervals[taskID]) return; // Если уже опрашивается, ничего не делаем

        pollingIntervals[taskID] = setInterval(async () => {
            try {
                const response = await fetch(`/image/${taskID}`);
                if (!response.ok) return; // Пропускаем, если ошибка (например, 404)

                const data = await response.json();
                const task = data.task;

                renderTask(task);

                // Если задача завершена (успешно или с ошибкой), прекращаем опрос
                if (task.status === 'COMPLETE' || task.status === 'FAILED') {
                    clearInterval(pollingIntervals[taskID]);
                    delete pollingIntervals[taskID];
                }
            } catch (error) {
                console.error(`Error polling task ${taskID}:`, error);
                clearInterval(pollingIntervals[taskID]);
                delete pollingIntervals[taskID];
            }
        }, 2000); // Опрашиваем каждые 2 секунды
    }

    // Функция отрисовки карточки задачи
    function renderTask(task) {
        let card = document.getElementById(`task-${task.id}`);
        if (!card) {
            if (tasksContainer.querySelector('p')) {
                tasksContainer.innerHTML = ''; // Убираем сообщение "Загрузите изображение"
            }
            card = document.createElement('div');
            card.id = `task-${task.id}`;
            card.className = 'task-card';
            tasksContainer.prepend(card); // Добавляем новые задачи в начало
        }

        const statusClass = `status-${task.status.toLowerCase()}`;

        let resultsHTML = '<p>В обработке...</p>';
        if (task.status === 'COMPLETE') {
            const filename = getFilename(task.original_path); // Получаем чистое имя файла, например, 'uuid.jpg'

            resultsHTML = `
            <div class="image-results">
                <div>
                    <h4>Оригинал</h4>
                    <a href="/${task.original_path}" target="_blank">
                       <img src="/${task.original_path}" alt="Original">
                    </a>
                </div>
                <div>
                    <h4>Resize</h4>
                    <img src="/processed/resize/${filename}" alt="Resized">
                </div>
                <div>
                    <h4>Thumbnail</h4>
                    <img src="/processed/thumbnail/${filename}" alt="Thumbnail">
                </div>
                 <div>
                    <h4>Watermark</h4>
                    <img src="/processed/watermark/${filename}" alt="Watermarked">
                </div>
            </div>
        `;
        } else if (task.status === 'FAILED') {
            resultsHTML = '<p>Произошла ошибка при обработке.</p>';
        }

        card.innerHTML = `
            <div class="task-header">
                <span class="task-id">ID: ${task.id}</span>
                <span class="task-status ${statusClass}">${task.status}</span>
            </div>
            ${resultsHTML}
            <button class="delete-btn" data-id="${task.id}" title="Удалить">×</button>
        `;

        // Добавляем обработчик на кнопку удаления
        card.querySelector('.delete-btn').addEventListener('click', (e) => {
            const idToDelete = e.target.getAttribute('data-id');
            if (confirm(`Вы уверены, что хотите удалить задачу ${idToDelete}?`)) {
                deleteTask(idToDelete);
            }
        });
    }

    // Функция удаления задачи
    async function deleteTask(taskID) {
        try {
            const response = await fetch(`/image/${taskID}`, { method: 'DELETE' });
            if (!response.ok && response.status !== 204) {
                throw new Error('Ошибка при удалении на сервере.');
            }

            // Удаляем из DOM
            const card = document.getElementById(`task-${taskID}`);
            if (card) card.remove();

            // Удаляем из localStorage
            removeTask(taskID);

            // Останавливаем опрос, если он был
            if (pollingIntervals[taskID]) {
                clearInterval(pollingIntervals[taskID]);
                delete pollingIntervals[taskID];
            }

        } catch (error) {
            console.error(`Failed to delete task ${taskID}:`, error);
            alert(error.message);
        }
    }

    // --- Функции для работы с localStorage ---
    function getTasks() {
        return JSON.parse(localStorage.getItem('imageTasks') || '[]');
    }

    function saveTask(taskID) {
        const tasks = getTasks();
        if (!tasks.includes(taskID)) {
            tasks.unshift(taskID); // Добавляем в начало
            localStorage.setItem('imageTasks', JSON.stringify(tasks));
        }
    }

    function removeTask(taskID) {
        let tasks = getTasks();
        tasks = tasks.filter(id => id !== taskID);
        localStorage.setItem('imageTasks', JSON.stringify(tasks));
        if (tasks.length === 0) {
            tasksContainer.innerHTML = '<p>Загрузите изображение, чтобы увидеть его здесь.</p>';
        }
    }

    function loadInitialTasks() {
        const tasks = getTasks();
        if (tasks.length > 0) {
            tasksContainer.innerHTML = ''; // Очищаем контейнер
            tasks.forEach(taskID => {
                // Сначала создаем заглушку, потом опрашиваем
                renderTask({ id: taskID, status: 'PROCESSING' });
                pollStatus(taskID);
            });
        }
    }

    // Вспомогательная функция для получения имени файла из пути
    function getFilename(path) {
        return path.split('/').pop();
    }
});