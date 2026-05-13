// Session ID — one per browser tab
const urlParams = new URLSearchParams(window.location.search);
let SESSION_ID = urlParams.get('session_id');
if (!SESSION_ID) {
  SESSION_ID = crypto.randomUUID();
  window.history.replaceState({}, '', `/?session_id=${SESSION_ID}`);
}

// ---- Toast Notifications ----

function showToast(msg, type) {
  const container = document.getElementById("toast-container");
  if (!container) return;

  const colors = {
    success: "bg-emerald-600",
    error: "bg-red-600",
    info: "bg-blue-600",
    warning: "bg-amber-600",
  };

  const toast = document.createElement("div");
  toast.className = `toast-enter flex items-center gap-2 px-4 py-3 rounded-lg shadow-lg text-white text-sm ${colors[type] || colors.info}`;
  toast.textContent = msg;
  container.appendChild(toast);

  setTimeout(() => {
    toast.classList.remove("toast-enter");
    toast.classList.add("toast-exit");
    toast.addEventListener("animationend", () => toast.remove());
  }, 4000);
}

// ---- Chat ----

function scrollToBottom() {
  const chat = document.getElementById("chat-messages");
  if (chat) chat.scrollTop = chat.scrollHeight;
}

function appendMessage(role, content) {
  const chat = document.getElementById("chat-messages");
  if (!chat) return null;

  const wrapper = document.createElement("div");
  wrapper.className = role === "user"
    ? "flex justify-end"
    : "flex justify-start";

  const bubble = document.createElement("div");
  bubble.className = role === "user"
    ? "chat-content max-w-[75%] px-4 py-3 rounded-2xl bg-blue-600 text-white rounded-br-md"
    : "chat-content max-w-[75%] px-4 py-3 rounded-2xl bg-gray-700 text-gray-100 rounded-bl-md";

  bubble.innerHTML = escapeAndFormat(content);
  wrapper.appendChild(bubble);
  chat.appendChild(wrapper);
  scrollToBottom();
  return bubble;
}

function escapeAndFormat(text) {
  const escaped = text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
  // Basic: convert newlines to <br>
  return escaped.replace(/\n/g, "<br>");
}

// ---- SSE Chat Submission ----

function submitChat() {
  const input = document.getElementById("chat-input");
  const prompt = input.value.trim();
  if (!prompt) return;

  input.value = "";
  appendMessage("user", prompt);

  const assistantBubble = appendMessage("assistant", "");
  assistantBubble.classList.add("streaming-cursor");

  const body = JSON.stringify({
    session_id: SESSION_ID,
    user_prompt: prompt,
    system_prompt: {},
  });

  fetch("/api/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json", "Accept": "text/event-stream" },
    body: body,
  }).then((response) => {
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let sseEvent = "message";
    let sseData = [];

    function read() {
      reader.read().then(({ done, value }) => {
        if (done) {
          assistantBubble.classList.remove("streaming-cursor");
          return;
        }

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop(); // keep incomplete line in buffer

        for (const line of lines) {
          if (line.startsWith("event: ")) {
            sseEvent = line.substring(7).trim();
          } else if (line.startsWith("data: ")) {
            sseData.push(line.substring(6));
          } else if (line.trim() === "") {
            // Empty line means end of event block
            if (sseData.length > 0 || sseEvent !== "message") {
              handleSSE(sseEvent, sseData.join("\n"), assistantBubble);
              sseEvent = "message";
              sseData = [];
            }
          }
        }

        read();
      });
    }

    read();
  }).catch((err) => {
    assistantBubble.classList.remove("streaming-cursor");
    showToast("Connection error: " + err.message, "error");
  });
}

function handleSSE(eventType, data, bubble) {
  switch (eventType) {
    case "message":
      let htmlToAppend = escapeAndFormat(data);

      const rateLimitMatch = data.match(/\[Rate limited\. Retrying in (\d+)s\.\.\.\]/);
      if (rateLimitMatch) {
        const secs = parseInt(rateLimitMatch[1], 10);
        const spanId = "retry-" + Date.now();
        
        // Replace the escaped string with actual HTML containing our span
        const escapedOriginal = escapeAndFormat(rateLimitMatch[0]);
        const replacementHtml = `[Rate limited. Retrying in <span id="${spanId}" class="text-amber-500 font-bold">${secs}</span>s...]`;
        
        htmlToAppend = htmlToAppend.replace(escapedOriginal, replacementHtml);
        
        // Start countdown
        let remaining = secs;
        const interval = setInterval(() => {
          remaining--;
          const span = document.getElementById(spanId);
          if (span && remaining >= 0) {
            span.textContent = remaining;
          } else {
            clearInterval(interval);
            if (span) span.classList.remove("text-amber-500");
          }
        }, 1000);
      }

      bubble.innerHTML += htmlToAppend;
      scrollToBottom();
      break;

    case "done":
      bubble.classList.remove("streaming-cursor");
      break;

    case "error":
      bubble.classList.remove("streaming-cursor");
      showToast(data, "error");
      break;

    case "consent":
      try {
        const req = JSON.parse(data);
        showConsentDialog(req);
      } catch (e) {
        showToast("Bad consent request", "error");
      }
      break;
  }
}

// ---- Consent Dialog ----

function showConsentDialog(req) {
  const modal = document.getElementById("consent-modal");
  const toolName = document.getElementById("consent-tool-name");
  const subject = document.getElementById("consent-subject");
  const details = document.getElementById("consent-details");

  if (!modal) return;

  toolName.textContent = req.tool;
  subject.textContent = req.subject;
  details.textContent = JSON.stringify(req.args, null, 2);

  modal.dataset.consentId = req.id;
  modal.classList.remove("hidden");
}

function respondConsent(action) {
  const modal = document.getElementById("consent-modal");
  const consentId = modal.dataset.consentId;

  modal.classList.add("hidden");

  fetch("/api/consent", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id: consentId, action: action }),
  }).then((resp) => {
    if (!resp.ok) {
      showToast("Consent response failed", "error");
    }
  }).catch((err) => {
    showToast("Consent error: " + err.message, "error");
  });
}

// ---- Keyboard shortcuts ----

document.addEventListener("DOMContentLoaded", () => {
  const input = document.getElementById("chat-input");
  if (input) {
    input.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        submitChat();
      }
    });
  }
});

// ---- Context Menu ----

function toggleMenu() {
  const menu = document.getElementById("menu-dropdown");
  if (menu.classList.contains("hidden")) {
    menu.classList.remove("hidden");
    document.addEventListener("click", closeMenuOnClickOutside);
  } else {
    menu.classList.add("hidden");
    document.removeEventListener("click", closeMenuOnClickOutside);
  }
}

function closeMenuOnClickOutside(e) {
  const menu = document.getElementById("menu-dropdown");
  const btn = document.getElementById("menu-btn");
  if (!menu.contains(e.target) && !btn.contains(e.target)) {
    menu.classList.add("hidden");
    document.removeEventListener("click", closeMenuOnClickOutside);
  }
}

function loadSession(sessionId) {
  // Simple navigation for now, later could be HTMX swap
  window.location.href = `/?session_id=${sessionId}`;
}

// ---- Model Management ----

async function updateModel() {
  const select = document.getElementById("model-select");
  if (!select) return;
  const modelId = select.value;
  
  try {
    const resp = await fetch("/api/settings");
    if (!resp.ok) throw new Error("Could not load settings");
    const cfg = await resp.json();
    cfg.activeModel = modelId;
    
    const saveResp = await fetch("/api/settings", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(cfg),
    });
    
    if (saveResp.ok) {
      showToast(`Model switched to ${modelId}`, "success");
    } else {
      throw new Error("Failed to save settings");
    }
  } catch (err) {
    showToast("Failed to update model: " + err.message, "error");
  }
}
