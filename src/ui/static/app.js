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

function appendMessage(role, content, model, timestamp) {
  const chat = document.getElementById("chat-messages");
  if (!chat) return null;

  const timeStr = timestamp || new Date().toISOString();
  const displayTime = new Date(timeStr).toLocaleTimeString();

  const wrapper = document.createElement("div");
  wrapper.className = role === "user"
    ? "flex flex-col items-end group"
    : "flex flex-col items-start group";

  const bubbleContainer = document.createElement("div");
  bubbleContainer.className = "relative max-w-[75%]";

  const bubble = document.createElement("div");
  bubble.className = role === "user"
    ? "chat-content px-4 py-3 rounded-2xl bg-blue-600 text-white rounded-br-md whitespace-pre-wrap"
    : "chat-content px-4 py-3 rounded-2xl bg-gray-700 text-gray-100 rounded-bl-md whitespace-pre-wrap";

  bubble.innerHTML = escapeAndFormat(content);
  bubbleContainer.appendChild(bubble);

  // Actions
  const actions = document.createElement("div");
  actions.className = "absolute -bottom-2 -right-2 flex items-center gap-1 z-10";

  // Edit Button
  const msgIndex = chat.querySelectorAll(".chat-content").length;
  const editBtn = document.createElement("button");
  editBtn.className = "w-5 h-5 bg-gray-800 border border-gray-700 rounded-full flex items-center justify-center text-[10px] text-gray-400 hover:border-emerald-500 hover:text-emerald-400 shadow-lg transition-all";
  editBtn.title = "Edit & Branch";
  editBtn.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>`;
  editBtn.onclick = (e) => editMessage(editBtn, msgIndex, role, model);
  actions.appendChild(editBtn);

  // Info Icon
  const infoIcon = document.createElement("div");
  infoIcon.className = "w-5 h-5 bg-gray-800 border border-gray-700 rounded-full flex items-center justify-center text-[10px] text-gray-400 cursor-help transition-all shadow-lg hover:border-blue-500 hover:text-blue-400 group/info";
  infoIcon.innerHTML = "i";
  
  // Tooltip
  const tooltip = document.createElement("div");
  tooltip.className = "absolute bottom-full right-0 mb-2 w-48 bg-gray-900 text-white text-[10px] p-2 rounded-lg border border-gray-700 shadow-2xl pointer-events-none opacity-0 group-hover/info:opacity-100 transition-opacity z-20";
  let tooltipHtml = `<div>🕒 ${displayTime}</div>`;
  if (model) tooltipHtml += `<div class="mt-1 border-t border-gray-800 pt-1">🤖 ${model}</div>`;
  tooltip.innerHTML = tooltipHtml;
  
  infoIcon.appendChild(tooltip);
  actions.appendChild(infoIcon);
  bubbleContainer.appendChild(actions);
  wrapper.appendChild(bubbleContainer);

  if (role === "assistant" && model) {
    const badge = document.createElement("div");
    badge.className = "text-[10px] text-gray-500 mt-1 ml-2 flex items-center gap-1 opacity-70";
    badge.innerHTML = `<span>🤖</span> ${model}`;
    wrapper.appendChild(badge);
  }

  chat.appendChild(wrapper);
  scrollToBottom();
  return bubble;
}

function escapeAndFormat(text) {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

// ---- SSE Chat Submission ----

function submitChat() {
  const input = document.getElementById("chat-input");
  const prompt = input.value.trim();
  if (!prompt) return;

  input.value = "";
  const now = new Date().toISOString();
  appendMessage("user", prompt, null, now);

  const assistantBubble = appendMessage("assistant", "", null, now);
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
      let content = data;
      let model = "";
      let timestamp = "";
      try {
        const parsed = JSON.parse(data);
        content = parsed.content;
        model = parsed.model;
        timestamp = parsed.timestamp;
      } catch (e) {
        // Fallback for non-JSON
      }

      let htmlToAppend = escapeAndFormat(content);

      const rateLimitMatch = content.match(/\[Rate limited\. Retrying in (\d+)s\.\.\.\]/);
      if (rateLimitMatch) {
        const secs = parseInt(rateLimitMatch[1], 10);
        const spanId = "retry-" + Date.now();
        
        const escapedOriginal = escapeAndFormat(rateLimitMatch[0]);
        const replacementHtml = `[Rate limited. Retrying in <span id="${spanId}" class="text-amber-500 font-bold">${secs}</span>s...]`;
        
        htmlToAppend = htmlToAppend.replace(escapedOriginal, replacementHtml);
        
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
      
      // If model is present and not already shown, we could add it.
      // But for streaming chunks, we only want to add it once.
      // Let's check if the badge already exists in the wrapper.
      if (model) {
        const wrapper = bubble.closest('.group');
        const tooltip = wrapper ? wrapper.querySelector('.group-hover\\/info\\:opacity-100') : null;
        if (tooltip && !tooltip.innerHTML.includes('🤖')) {
          tooltip.innerHTML += `<div class="mt-1 border-t border-gray-800 pt-1">🤖 ${model}</div>`;
        }
        
        if (wrapper && !wrapper.querySelector('.model-badge')) {
          const badge = document.createElement("div");
          badge.className = "model-badge text-[10px] text-gray-500 mt-1 ml-2 flex items-center gap-1 opacity-70";
          badge.innerHTML = `<span>🤖</span> ${model}`;
          wrapper.appendChild(badge);
        }
      }
      
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
  if (SESSION_ID) {
    document.getElementById('delete-chat-container')?.classList.remove('hidden');
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

function confirmDeleteChat() {
  fetch(`/ui/delete?session_id=${SESSION_ID}`, { method: 'POST' })
    .then(res => {
      if (res.ok) {
        window.location.href = '/';
      } else {
        showToast("Failed to delete chat", "error");
      }
    });
}

async function editMessage(btn, index, role, model) {
  const bubbleContainer = btn.closest(".relative.max-w-\\[75\\%\\]");
  const contentEl = bubbleContainer.querySelector(".chat-content");
  const originalText = contentEl.innerText;
  
  // Hide actions
  const actions = bubbleContainer.querySelector(".flex.items-center.gap-1.z-10");
  actions.classList.add("hidden");
  
  // Create editor
  const editor = document.createElement("div");
  editor.className = "flex flex-col gap-2 mt-2";
  
  const textarea = document.createElement("textarea");
  textarea.className = "w-full bg-gray-800 border border-gray-600 rounded p-2 text-sm text-white focus:outline-none focus:border-emerald-500 min-h-[100px]";
  textarea.value = originalText;
  
  const btnRow = document.createElement("div");
  btnRow.className = "flex gap-2 justify-end";
  
  const saveBtn = document.createElement("button");
  saveBtn.className = "bg-emerald-600 hover:bg-emerald-500 text-white text-[10px] px-3 py-1 rounded transition-colors";
  saveBtn.textContent = "Branch & Save";
  
  const cancelBtn = document.createElement("button");
  cancelBtn.className = "bg-gray-700 hover:bg-gray-600 text-gray-300 text-[10px] px-3 py-1 rounded transition-colors";
  cancelBtn.textContent = "Cancel";
  
  btnRow.appendChild(cancelBtn);
  btnRow.appendChild(saveBtn);
  editor.appendChild(textarea);
  editor.appendChild(btnRow);
  
  // Hide original content and show editor
  contentEl.classList.add("hidden");
  bubbleContainer.appendChild(editor);
  textarea.focus();
  
  cancelBtn.onclick = () => {
    editor.remove();
    contentEl.classList.remove("hidden");
    actions.classList.remove("hidden");
  };
  
  saveBtn.onclick = async () => {
    const newContent = textarea.value;
    if (newContent === originalText) {
       cancelBtn.onclick();
       return;
    }
    
    saveBtn.disabled = true;
    saveBtn.textContent = "Branching...";
    
    const newSessionId = crypto.randomUUID();
    try {
      const resp = await fetch("/ui/branch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          old_session_id: SESSION_ID,
          new_session_id: newSessionId,
          index: index,
          role: role,
          content: newContent,
          model: model || ""
        })
      });
      
      if (resp.ok) {
        // We set a flag in sessionStorage to show the toast after redirect
        sessionStorage.setItem('show_branched_toast', 'true');
        window.location.href = `/?session_id=${newSessionId}`;
      } else {
        throw new Error("Failed to branch");
      }
    } catch (err) {
      showToast("Failed to branch chat: " + err.message, "error");
      saveBtn.disabled = false;
      saveBtn.textContent = "Branch & Save";
    }
  };
}

// Check for branched toast on load
document.addEventListener("DOMContentLoaded", () => {
  if (sessionStorage.getItem('show_branched_toast')) {
    showToast("Chat branched", "success");
    sessionStorage.removeItem('show_branched_toast');
  }
});
