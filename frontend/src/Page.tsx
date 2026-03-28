import { useState, useRef, useCallback, useEffect, useReducer } from "react";

// ─── Font injection ────────────────────────────────────────────────────────────
const fontLink = document.createElement("link");
fontLink.rel = "stylesheet";
fontLink.href = "https://fonts.googleapis.com/css2?family=Syne:wght@400;600;700;800&family=DM+Sans:wght@300;400;500&display=swap";
document.head.appendChild(fontLink);

// ─── Component Definitions (the palette) ──────────────────────────────────────
const COMPONENT_DEFS = {
	// Layout
	section: { category: "Layout", label: "Section", icon: "▣", defaultW: 700, defaultH: 180, minW: 200, minH: 60, color: "#6366f1" },
	container: { category: "Layout", label: "Container", icon: "◫", defaultW: 400, defaultH: 140, minW: 120, minH: 40, color: "#8b5cf6" },
	columns: { category: "Layout", label: "2 Columns", icon: "⊞", defaultW: 460, defaultH: 130, minW: 200, minH: 60, color: "#a78bfa" },
	// Form
	input: { category: "Form", label: "Text Input", icon: "▭", defaultW: 260, defaultH: 56, minW: 120, minH: 40, color: "#06b6d4" },
	textarea: { category: "Form", label: "Textarea", icon: "▬", defaultW: 280, defaultH: 100, minW: 120, minH: 60, color: "#0ea5e9" },
	select: { category: "Form", label: "Dropdown", icon: "⌄", defaultW: 220, defaultH: 48, minW: 100, minH: 36, color: "#38bdf8" },
	checkbox: { category: "Form", label: "Checkbox", icon: "☑", defaultW: 180, defaultH: 40, minW: 100, minH: 32, color: "#22d3ee" },
	radio: { category: "Form", label: "Radio Group", icon: "◉", defaultW: 180, defaultH: 40, minW: 100, minH: 32, color: "#67e8f9" },
	// Content
	heading: { category: "Content", label: "Heading", icon: "H", defaultW: 300, defaultH: 56, minW: 80, minH: 32, color: "#f59e0b" },
	text: { category: "Content", label: "Paragraph", icon: "¶", defaultW: 320, defaultH: 80, minW: 80, minH: 32, color: "#fbbf24" },
	image: { category: "Content", label: "Image", icon: "⬜", defaultW: 260, defaultH: 160, minW: 60, minH: 40, color: "#f97316" },
	divider: { category: "Content", label: "Divider", icon: "─", defaultW: 280, defaultH: 24, minW: 60, minH: 16, color: "#fb923c" },
	// Action
	button: { category: "Action", label: "Button", icon: "⬡", defaultW: 140, defaultH: 44, minW: 80, minH: 32, color: "#10b981" },
	submit: { category: "Action", label: "Submit", icon: "➤", defaultW: 160, defaultH: 44, minW: 80, minH: 32, color: "#34d399" },
};

const CATEGORIES = ["Layout", "Form", "Content", "Action"];

// ─── Default props per type ────────────────────────────────────────────────────
const defaultProps = (type) => ({
	heading: { text: "Heading Text", level: "h2", fontSize: 28, fontWeight: 700, color: "#111827", align: "left" },
	text: { text: "Add your paragraph text here. Click to edit.", fontSize: 15, color: "#374151", align: "left", lineHeight: 1.6 },
	button: { label: "Click me", variant: "primary", radius: 8 },
	submit: { label: "Submit Form", variant: "primary", radius: 8 },
	input: { placeholder: "Enter text...", label: "Label", required: false, type: "text" },
	textarea: { placeholder: "Your message...", label: "Message", rows: 3 },
	select: { label: "Select option", options: "Option 1\nOption 2\nOption 3" },
	checkbox: { label: "I agree to terms", checked: false },
	radio: { label: "Choose one", options: "Option A\nOption B" },
	image: { src: "", alt: "Image", objectFit: "cover", radius: 8 },
	divider: { color: "#e5e7eb", thickness: 1 },
	section: { bg: "#f9fafb", padding: 24, radius: 12, label: "Section" },
	container: { bg: "#ffffff", padding: 16, radius: 8, label: "Container", border: true },
	columns: { bg: "transparent", padding: 12, radius: 8, label: "2 Columns", gap: 16 },
}[type] || {});

// ─── uid ──────────────────────────────────────────────────────────────────────
let _id = 1;
const uid = () => `el-${_id++}`;

// ─── Reducer ──────────────────────────────────────────────────────────────────
function builderReducer(state, action) {
	switch (action.type) {
		case "ADD_NODE": {
			const { node } = action;
			const next = { ...state.nodes, [node.id]: node };
			if (node.parentId) {
				next[node.parentId] = { ...next[node.parentId], children: [...(next[node.parentId].children || []), node.id] };
			}
			return {
				...state,
				nodes: next,
				rootOrder: node.parentId ? state.rootOrder : [...state.rootOrder, node.id],
			};
		}
		case "UPDATE_NODE":
			return { ...state, nodes: { ...state.nodes, [action.id]: { ...state.nodes[action.id], ...action.patch } } };
		case "UPDATE_PROPS":
			return { ...state, nodes: { ...state.nodes, [action.id]: { ...state.nodes[action.id], props: { ...state.nodes[action.id].props, ...action.patch } } } };
		case "DELETE_NODE": {
			const collectIds = (id, nodes) => {
				const n = nodes[id];
				if (!n) return [];
				return [id, ...(n.children || []).flatMap(c => collectIds(c, nodes))];
			};
			const toRemove = new Set(collectIds(action.id, state.nodes));
			const next = Object.fromEntries(Object.entries(state.nodes).filter(([k]) => !toRemove.has(k)));
			// remove from parent
			const node = state.nodes[action.id];
			if (node?.parentId && next[node.parentId]) {
				next[node.parentId] = { ...next[node.parentId], children: next[node.parentId].children.filter(c => !toRemove.has(c)) };
			}
			return { ...state, nodes: next, rootOrder: state.rootOrder.filter(id => !toRemove.has(id)) };
		}
		case "MOVE_NODE": {
			const { id, x, y } = action;
			return { ...state, nodes: { ...state.nodes, [id]: { ...state.nodes[id], x, y } } };
		}
		case "RESIZE_NODE": {
			const { id, x, y, w, h } = action;
			return { ...state, nodes: { ...state.nodes, [id]: { ...state.nodes[id], x, y, w, h } } };
		}
		default: return state;
	}
}

const initState = () => {
	const s1 = uid();
	const h1 = uid();
	const t1 = uid();
	const b1 = uid();
	const inp1 = uid();
	const inp2 = uid();
	const sub1 = uid();
	return {
		rootOrder: [s1],
		nodes: {
			[s1]: { id: s1, type: "section", parentId: null, x: 40, y: 40, w: 680, h: 480, children: [h1, t1, inp1, inp2, b1, sub1], props: { ...defaultProps("section"), label: "Contact Form" } },
			[h1]: { id: h1, type: "heading", parentId: s1, x: 32, y: 32, w: 380, h: 52, children: [], props: { ...defaultProps("heading"), text: "Get in Touch" } },
			[t1]: { id: t1, type: "text", parentId: s1, x: 32, y: 96, w: 420, h: 56, children: [], props: { ...defaultProps("text"), text: "Fill out the form below and we'll get back to you shortly." } },
			[inp1]: { id: inp1, type: "input", parentId: s1, x: 32, y: 168, w: 280, h: 56, children: [], props: { ...defaultProps("input"), label: "Full Name", placeholder: "John Doe" } },
			[inp2]: { id: inp2, type: "input", parentId: s1, x: 336, y: 168, w: 280, h: 56, children: [], props: { ...defaultProps("input"), label: "Email", placeholder: "you@example.com", type: "email" } },
			[b1]: { id: b1, type: "textarea", parentId: s1, x: 32, y: 248, w: 584, h: 100, children: [], props: { ...defaultProps("textarea"), label: "Message" } },
			[sub1]: { id: sub1, type: "submit", parentId: s1, x: 32, y: 376, w: 160, h: 44, children: [], props: { ...defaultProps("submit") } },
		}
	};
};

// ─── Main App ─────────────────────────────────────────────────────────────────
export default function App() {
	const [state, dispatch] = useReducer(builderReducer, null, initState);
	const [selected, setSelected] = useState(null);
	const [draggingPalette, setDraggingPalette] = useState(null);
	const [zoom, setZoom] = useState(1);
	const [preview, setPreview] = useState(false);
	const canvasRef = useRef(null);
	const dragState = useRef(null);

	const selectedNode = selected ? state.nodes[selected] : null;
	const def = selectedNode ? COMPONENT_DEFS[selectedNode.type] : null;

	// ── palette drag ──
	const onPaletteDragStart = (e, type) => {
		setDraggingPalette(type);
		e.dataTransfer.effectAllowed = "copy";
		e.dataTransfer.setData("component-type", type);
	};

	const onCanvasDrop = (e) => {
		e.preventDefault();
		const type = e.dataTransfer.getData("component-type");
		if (!type || !COMPONENT_DEFS[type]) return;
		const rect = canvasRef.current.getBoundingClientRect();
		const x = (e.clientX - rect.left) / zoom - COMPONENT_DEFS[type].defaultW / 2;
		const y = (e.clientY - rect.top) / zoom - COMPONENT_DEFS[type].defaultH / 2;
		const node = {
			id: uid(), type,
			parentId: null,
			x: Math.max(0, x), y: Math.max(0, y),
			w: COMPONENT_DEFS[type].defaultW,
			h: COMPONENT_DEFS[type].defaultH,
			children: [],
			props: defaultProps(type),
		};
		dispatch({ type: "ADD_NODE", node });
		setSelected(node.id);
		setDraggingPalette(null);
	};

	// ── canvas drag/resize ──
	const startMove = (e, id) => {
		if (e.button !== 0) return;
		e.stopPropagation();
		const node = state.nodes[id];
		dragState.current = { op: "move", id, sx: e.clientX, sy: e.clientY, ox: node.x, oy: node.y };
		setSelected(id);
	};

	const startResize = (e, id, handle) => {
		e.stopPropagation(); e.preventDefault();
		const node = state.nodes[id];
		dragState.current = { op: "resize", id, handle, sx: e.clientX, sy: e.clientY, ox: node.x, oy: node.y, ow: node.w, oh: node.h };
	};

	useEffect(() => {
		const onMove = (e) => {
			const ds = dragState.current;
			if (!ds) return;
			const dx = (e.clientX - ds.sx) / zoom;
			const dy = (e.clientY - ds.sy) / zoom;
			const node = state.nodes[ds.id];
			const def = COMPONENT_DEFS[node.type];
			if (ds.op === "move") {
				let nx = ds.ox + dx, ny = ds.oy + dy;
				if (node.parentId) {
					const p = state.nodes[node.parentId];
					nx = Math.max(0, Math.min(nx, p.w - node.w));
					ny = Math.max(0, Math.min(ny, p.h - node.h));
				}
				dispatch({ type: "MOVE_NODE", id: ds.id, x: nx, y: ny });
			} else if (ds.op === "resize") {
				const { handle, ox, oy, ow, oh } = ds;
				let nx = ox, ny = oy, nw = ow, nh = oh;
				if (handle.includes("e")) nw = Math.max(def.minW, ow + dx);
				if (handle.includes("s")) nh = Math.max(def.minH, oh + dy);
				if (handle.includes("w")) { nw = Math.max(def.minW, ow - dx); nx = ox + ow - nw; }
				if (handle.includes("n")) { nh = Math.max(def.minH, oh - dy); ny = oy + oh - nh; }
				dispatch({ type: "RESIZE_NODE", id: ds.id, x: nx, y: ny, w: nw, h: nh });
			}
		};
		const onUp = () => { dragState.current = null; };
		window.addEventListener("mousemove", onMove);
		window.addEventListener("mouseup", onUp);
		return () => { window.removeEventListener("mousemove", onMove); window.removeEventListener("mouseup", onUp); };
	}, [state.nodes, zoom]);

	const deleteSelected = () => {
		if (!selected) return;
		dispatch({ type: "DELETE_NODE", id: selected });
		setSelected(null);
	};

	useEffect(() => {
		const handler = (e) => { if (e.key === "Delete" || e.key === "Backspace") { if (document.activeElement === document.body) deleteSelected(); } };
		window.addEventListener("keydown", handler);
		return () => window.removeEventListener("keydown", handler);
	}, [selected]);

	return (
		<div style={S.root}>
			<style>{CSS}</style>
			{/* ── Topbar ── */}
			<header style={S.topbar}>
				<div style={S.topbarLeft}>
					<span style={S.brand}>⬡ Forma</span>
					<span style={S.sep}>/</span>
					<span style={S.docName}>Untitled Page</span>
				</div>
				<div style={S.topbarCenter}>
					<button style={{ ...S.zoomBtn, ...(zoom === 0.75 ? S.zoomBtnActive : {}) }} onClick={() => setZoom(0.75)}>75%</button>
					<button style={{ ...S.zoomBtn, ...(zoom === 1 ? S.zoomBtnActive : {}) }} onClick={() => setZoom(1)}>100%</button>
					<button style={{ ...S.zoomBtn, ...(zoom === 1.25 ? S.zoomBtnActive : {}) }} onClick={() => setZoom(1.25)}>125%</button>
				</div>
				<div style={S.topbarRight}>
					<button style={{ ...S.topBtn, ...(preview ? S.topBtnActive : {}) }} onClick={() => setPreview(p => !p)}>
						{preview ? "✏ Edit" : "▶ Preview"}
					</button>
					<button style={{ ...S.topBtn, ...S.topBtnPrimary }}>Publish</button>
				</div>
			</header>

			<div style={S.workspace}>
				{/* ── Left Panel ── */}
				{!preview && (
					<aside style={S.leftPanel}>
						<div style={S.panelHeader}>Components</div>
						{CATEGORIES.map(cat => (
							<div key={cat}>
								<div style={S.catLabel}>{cat}</div>
								<div style={S.paletteGrid}>
									{Object.entries(COMPONENT_DEFS)
										.filter(([, d]) => d.category === cat)
										.map(([type, d]) => (
											<div
												key={type}
												draggable
												onDragStart={(e) => onPaletteDragStart(e, type)}
												onDragEnd={() => setDraggingPalette(null)}
												style={{ ...S.paletteItem, borderColor: draggingPalette === type ? d.color : "#e5e7eb" }}
												title={d.label}
											>
												<span style={{ ...S.paletteIcon, color: d.color }}>{d.icon}</span>
												<span style={S.paletteLabel}>{d.label}</span>
											</div>
										))}
								</div>
							</div>
						))}

						{/* Layers */}
						<div style={{ marginTop: 20 }}>
							<div style={S.panelHeader}>Layers</div>
							{state.rootOrder.map(id => (
								<LayerItem
									key={id}
									id={id}
									nodes={state.nodes}
									selected={selected}
									onSelect={setSelected}
									depth={0}
								/>
							))}
						</div>
					</aside>
				)}

				{/* ── Canvas ── */}
				<main style={S.canvasWrap}>
					<div
						style={S.canvasScroll}
						onDragOver={e => e.preventDefault()}
						onDrop={onCanvasDrop}
						onClick={() => setSelected(null)}
					>
						<div
							ref={canvasRef}
							style={{ ...S.canvas, transform: `scale(${zoom})`, transformOrigin: "top left" }}
						>
							{/* Page background */}
							<div style={S.page} onClick={e => e.stopPropagation()}>
								{state.rootOrder.map(id => state.nodes[id] && (
									<CanvasNode
										key={id}
										id={id}
										nodes={state.nodes}
										selected={selected}
										onSelect={setSelected}
										onStartMove={startMove}
										onStartResize={startResize}
										preview={preview}
										depth={0}
									/>
								))}
								{state.rootOrder.length === 0 && (
									<div style={S.emptyState}>
										<div style={S.emptyIcon}>⬡</div>
										<div style={S.emptyTitle}>Drag components here</div>
										<div style={S.emptyText}>Start building by dragging components from the left panel</div>
									</div>
								)}
							</div>
						</div>
					</div>
				</main>

				{/* ── Right Panel ── */}
				{!preview && selectedNode && (
					<aside style={S.rightPanel}>
						<div style={S.propHeader}>
							<span style={{ ...S.propTypeTag, background: def?.color + "22", color: def?.color }}>
								{def?.icon} {def?.label}
							</span>
							<button style={S.deleteBtn} onClick={deleteSelected} title="Delete (Del)">✕</button>
						</div>
						<PropertiesPanel
							node={selectedNode}
							onUpdate={(patch) => dispatch({ type: "UPDATE_NODE", id: selectedNode.id, patch })}
							onUpdateProps={(patch) => dispatch({ type: "UPDATE_PROPS", id: selectedNode.id, patch })}
						/>
					</aside>
				)}
				{!preview && !selectedNode && (
					<aside style={S.rightPanel}>
						<div style={S.noSelection}>
							<span style={S.noSelIcon}>◎</span>
							<span>Select an element<br />to edit its properties</span>
						</div>
					</aside>
				)}
			</div>
		</div>
	);
}

// ─── Canvas Node ──────────────────────────────────────────────────────────────
function CanvasNode({ id, nodes, selected, onSelect, onStartMove, onStartResize, preview, depth }) {
	const node = nodes[id];
	if (!node) return null;
	const def = COMPONENT_DEFS[node.type];
	const isSelected = selected === id;
	const isContainer = ["section", "container", "columns"].includes(node.type);

	return (
		<div
			style={{
				position: "absolute",
				left: node.x, top: node.y,
				width: node.w, height: node.h,
				outline: !preview && isSelected ? `2px solid ${def.color}` : !preview ? "1px dashed transparent" : "none",
				outlineOffset: 1,
				borderRadius: 4,
				zIndex: isSelected ? 50 : 10 + depth,
				cursor: preview ? "default" : "default",
				boxSizing: "border-box",
			}}
			onMouseEnter={e => { if (!preview) e.currentTarget.style.outline = isSelected ? `2px solid ${def.color}` : `1px dashed ${def.color}88`; }}
			onMouseLeave={e => { if (!preview) e.currentTarget.style.outline = isSelected ? `2px solid ${def.color}` : "1px dashed transparent"; }}
			onClick={e => { e.stopPropagation(); onSelect(id); }}
		>
			{/* Drag handle */}
			{!preview && (
				<div
					style={{ ...S.dragHandle, background: isSelected ? def.color : "transparent" }}
					onMouseDown={e => onStartMove(e, id)}
					title="Drag to move"
				>
					{isSelected && <span style={S.dragHandleLabel}>{def.icon} {node.props?.label || def.label}</span>}
				</div>
			)}

			{/* Rendered Component */}
			<ComponentRenderer node={node} nodes={nodes} isContainer={isContainer} preview={preview}>
				{isContainer && (node.children || []).map(cid => nodes[cid] && (
					<CanvasNode
						key={cid}
						id={cid}
						nodes={nodes}
						selected={selected}
						onSelect={onSelect}
						onStartMove={onStartMove}
						onStartResize={onStartResize}
						preview={preview}
						depth={depth + 1}
					/>
				))}
			</ComponentRenderer>

			{/* Resize handles */}
			{!preview && isSelected && (
				<>
					{["n", "s", "e", "w", "ne", "nw", "se", "sw"].map(h => (
						<ResizeHandle key={h} handle={h} color={def.color} onMouseDown={e => { e.stopPropagation(); onStartResize(e, id, h); }} />
					))}
				</>
			)}
		</div>
	);
}

// ─── Component Renderer (the actual visual output) ────────────────────────────
function ComponentRenderer({ node, nodes, isContainer, preview, children }) {
	const p = node.props || {};
	const full = { width: "100%", height: "100%", boxSizing: "border-box" };

	switch (node.type) {
		case "section":
			return (
				<div style={{ ...full, background: p.bg || "#f9fafb", borderRadius: p.radius || 12, padding: p.padding || 24, position: "relative", overflow: "hidden" }}>
					{children}
					{!preview && React.Children.count(children) === 0 && (
						<div style={S.dropHint}>Drop components inside</div>
					)}
				</div>
			);
		case "container":
			return (
				<div style={{ ...full, background: p.bg || "#fff", borderRadius: p.radius || 8, padding: p.padding || 16, border: p.border ? "1px solid #e5e7eb" : "none", position: "relative", overflow: "hidden" }}>
					{children}
					{!preview && React.Children.count(children) === 0 && (
						<div style={S.dropHint}>Drop here</div>
					)}
				</div>
			);
		case "columns":
			return (
				<div style={{ ...full, display: "grid", gridTemplateColumns: "1fr 1fr", gap: p.gap || 16, padding: p.padding || 12, position: "relative", overflow: "hidden" }}>
					<div style={{ background: "#f1f5f9", borderRadius: 6, minHeight: 40, position: "relative" }}>{children}</div>
					<div style={{ background: "#f1f5f9", borderRadius: 6, minHeight: 40 }}></div>
				</div>
			);
		case "heading":
			const Tag = p.level || "h2";
			return <div style={{ ...full, display: "flex", alignItems: "center" }}><Tag style={{ margin: 0, fontSize: p.fontSize || 28, fontWeight: p.fontWeight || 700, color: p.color || "#111827", textAlign: p.align || "left", fontFamily: "'Syne', sans-serif", lineHeight: 1.2 }}>{p.text || "Heading"}</Tag></div>;
		case "text":
			return <p style={{ ...full, margin: 0, fontSize: p.fontSize || 15, color: p.color || "#374151", textAlign: p.align || "left", lineHeight: p.lineHeight || 1.6, fontFamily: "'DM Sans', sans-serif" }}>{p.text || "Paragraph"}</p>;
		case "button":
		case "submit":
			return (
				<div style={{ ...full, display: "flex", alignItems: "center", justifyContent: "flex-start" }}>
					<button
						style={{ padding: "10px 24px", background: p.variant === "outline" ? "transparent" : "#111827", color: p.variant === "outline" ? "#111827" : "#fff", border: p.variant === "outline" ? "2px solid #111827" : "none", borderRadius: p.radius || 8, fontFamily: "'DM Sans', sans-serif", fontWeight: 500, fontSize: 14, cursor: "pointer", letterSpacing: "0.02em" }}
					>{p.label || "Button"}</button>
				</div>
			);
		case "input":
			return (
				<div style={{ ...full, display: "flex", flexDirection: "column", justifyContent: "center", gap: 4 }}>
					{p.label && <label style={S.formLabel}>{p.label}{p.required && <span style={{ color: "#ef4444" }}> *</span>}</label>}
					<input type={p.type || "text"} placeholder={p.placeholder || ""} style={S.formInput} readOnly={!preview} />
				</div>
			);
		case "textarea":
			return (
				<div style={{ ...full, display: "flex", flexDirection: "column", justifyContent: "flex-start", gap: 4 }}>
					{p.label && <label style={S.formLabel}>{p.label}</label>}
					<textarea placeholder={p.placeholder || ""} rows={p.rows || 3} style={{ ...S.formInput, resize: "none", flex: 1 }} readOnly={!preview} />
				</div>
			);
		case "select":
			return (
				<div style={{ ...full, display: "flex", flexDirection: "column", justifyContent: "center", gap: 4 }}>
					{p.label && <label style={S.formLabel}>{p.label}</label>}
					<select style={S.formInput}>
						{(p.options || "").split("\n").map((o, i) => <option key={i}>{o}</option>)}
					</select>
				</div>
			);
		case "checkbox":
			return (
				<div style={{ ...full, display: "flex", alignItems: "center", gap: 8 }}>
					<input type="checkbox" defaultChecked={p.checked} style={{ width: 16, height: 16, accentColor: "#6366f1" }} />
					<span style={{ fontSize: 14, color: "#374151", fontFamily: "'DM Sans', sans-serif" }}>{p.label || "Checkbox"}</span>
				</div>
			);
		case "radio":
			return (
				<div style={{ ...full, display: "flex", flexDirection: "column", justifyContent: "center", gap: 6 }}>
					{p.label && <span style={S.formLabel}>{p.label}</span>}
					{(p.options || "").split("\n").map((o, i) => (
						<label key={i} style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 14, color: "#374151", fontFamily: "'DM Sans', sans-serif", cursor: "pointer" }}>
							<input type="radio" name={node.id} style={{ accentColor: "#6366f1" }} />{o}
						</label>
					))}
				</div>
			);
		case "image":
			return p.src
				? <img src={p.src} alt={p.alt || ""} style={{ width: "100%", height: "100%", objectFit: p.objectFit || "cover", borderRadius: p.radius || 8, display: "block" }} />
				: <div style={{ ...full, background: "#f1f5f9", borderRadius: p.radius || 8, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 6, color: "#94a3b8", fontSize: 13, fontFamily: "'DM Sans', sans-serif" }}>
					<span style={{ fontSize: 28 }}>🖼</span>Image Placeholder
				</div>;
		case "divider":
			return <div style={{ ...full, display: "flex", alignItems: "center" }}><hr style={{ width: "100%", border: "none", borderTop: `${p.thickness || 1}px solid ${p.color || "#e5e7eb"}`, margin: 0 }} /></div>;
		default:
			return <div style={{ ...full, background: "#f8fafc", display: "flex", alignItems: "center", justifyContent: "center", color: "#94a3b8", fontSize: 12 }}>{node.type}</div>;
	}
}

// ─── Resize Handle ────────────────────────────────────────────────────────────
function ResizeHandle({ handle, color, onMouseDown }) {
	const s = 8;
	const base = { position: "absolute", width: s, height: s, background: "#fff", border: `2px solid ${color}`, borderRadius: 2, zIndex: 100 };
	const pos = {
		n: { top: -s / 2, left: "50%", transform: "translateX(-50%)", cursor: "n-resize" },
		s: { bottom: -s / 2, left: "50%", transform: "translateX(-50%)", cursor: "s-resize" },
		e: { right: -s / 2, top: "50%", transform: "translateY(-50%)", cursor: "e-resize" },
		w: { left: -s / 2, top: "50%", transform: "translateY(-50%)", cursor: "w-resize" },
		ne: { top: -s / 2, right: -s / 2, cursor: "ne-resize" },
		nw: { top: -s / 2, left: -s / 2, cursor: "nw-resize" },
		se: { bottom: -s / 2, right: -s / 2, cursor: "se-resize" },
		sw: { bottom: -s / 2, left: -s / 2, cursor: "sw-resize" },
	}[handle];
	return <div style={{ ...base, ...pos }} onMouseDown={onMouseDown} />;
}

// ─── Properties Panel ─────────────────────────────────────────────────────────
function PropertiesPanel({ node, onUpdate, onUpdateProps }) {
	const p = node.props || {};
	const Field = ({ label, children }) => (
		<div style={S.field}>
			<div style={S.fieldLabel}>{label}</div>
			{children}
		</div>
	);
	const Input = ({ value, onChange, type = "text", ...rest }) => (
		<input type={type} value={value ?? ""} onChange={e => onChange(e.target.value)} style={S.propInput} {...rest} />
	);
	const NumberInput = ({ value, onChange, min, max }) => (
		<input type="number" value={value ?? 0} min={min} max={max} onChange={e => onChange(+e.target.value)} style={{ ...S.propInput, width: 72 }} />
	);
	const ColorInput = ({ value, onChange }) => (
		<div style={{ display: "flex", gap: 6, alignItems: "center" }}>
			<input type="color" value={value || "#000000"} onChange={e => onChange(e.target.value)} style={{ width: 32, height: 28, border: "1px solid #e5e7eb", borderRadius: 4, cursor: "pointer", padding: 2 }} />
			<Input value={value} onChange={onChange} />
		</div>
	);

	// Geometry
	return (
		<div style={S.propBody}>
			<div style={S.propSection}>
				<div style={S.propSectionTitle}>Layout</div>
				<div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
					<Field label="X"><NumberInput value={Math.round(node.x)} onChange={v => onUpdate({ x: v })} /></Field>
					<Field label="Y"><NumberInput value={Math.round(node.y)} onChange={v => onUpdate({ y: v })} /></Field>
					<Field label="Width"><NumberInput value={Math.round(node.w)} onChange={v => onUpdate({ w: v })} min={40} /></Field>
					<Field label="Height"><NumberInput value={Math.round(node.h)} onChange={v => onUpdate({ h: v })} min={20} /></Field>
				</div>
			</div>

			{/* Type-specific props */}
			{(node.type === "heading" || node.type === "text") && (
				<div style={S.propSection}>
					<div style={S.propSectionTitle}>Text</div>
					<Field label="Content">
						<textarea value={p.text || ""} onChange={e => onUpdateProps({ text: e.target.value })} style={{ ...S.propInput, height: 72, resize: "vertical" }} />
					</Field>
					<div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
						<Field label="Font Size"><NumberInput value={p.fontSize} onChange={v => onUpdateProps({ fontSize: v })} min={8} max={96} /></Field>
						{node.type === "heading" && <Field label="Font Weight"><NumberInput value={p.fontWeight} onChange={v => onUpdateProps({ fontWeight: v })} min={100} max={900} /></Field>}
					</div>
					<Field label="Color"><ColorInput value={p.color} onChange={v => onUpdateProps({ color: v })} /></Field>
					<Field label="Align">
						<div style={{ display: "flex", gap: 4 }}>
							{["left", "center", "right"].map(a => (
								<button key={a} onClick={() => onUpdateProps({ align: a })} style={{ ...S.alignBtn, ...(p.align === a ? S.alignBtnActive : {}) }}>{a[0].toUpperCase()}</button>
							))}
						</div>
					</Field>
				</div>
			)}

			{(node.type === "button" || node.type === "submit") && (
				<div style={S.propSection}>
					<div style={S.propSectionTitle}>Button</div>
					<Field label="Label"><Input value={p.label} onChange={v => onUpdateProps({ label: v })} /></Field>
					<Field label="Variant">
						<div style={{ display: "flex", gap: 4 }}>
							{["primary", "outline", "ghost"].map(v => (
								<button key={v} onClick={() => onUpdateProps({ variant: v })} style={{ ...S.alignBtn, ...(p.variant === v ? S.alignBtnActive : {}) }}>{v}</button>
							))}
						</div>
					</Field>
					<Field label="Border Radius"><NumberInput value={p.radius} onChange={v => onUpdateProps({ radius: v })} min={0} max={40} /></Field>
				</div>
			)}

			{(node.type === "input" || node.type === "textarea") && (
				<div style={S.propSection}>
					<div style={S.propSectionTitle}>Field</div>
					<Field label="Label"><Input value={p.label} onChange={v => onUpdateProps({ label: v })} /></Field>
					<Field label="Placeholder"><Input value={p.placeholder} onChange={v => onUpdateProps({ placeholder: v })} /></Field>
					{node.type === "input" && (
						<Field label="Type">
							<select value={p.type || "text"} onChange={e => onUpdateProps({ type: e.target.value })} style={S.propInput}>
								{["text", "email", "password", "number", "tel", "url"].map(t => <option key={t}>{t}</option>)}
							</select>
						</Field>
					)}
					<Field label="Required">
						<label style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 13 }}>
							<input type="checkbox" checked={!!p.required} onChange={e => onUpdateProps({ required: e.target.checked })} style={{ accentColor: "#6366f1" }} />
							Required field
						</label>
					</Field>
				</div>
			)}

			{node.type === "select" && (
				<div style={S.propSection}>
					<div style={S.propSectionTitle}>Select</div>
					<Field label="Label"><Input value={p.label} onChange={v => onUpdateProps({ label: v })} /></Field>
					<Field label="Options (one per line)">
						<textarea value={p.options || ""} onChange={e => onUpdateProps({ options: e.target.value })} style={{ ...S.propInput, height: 80, resize: "vertical" }} />
					</Field>
				</div>
			)}

			{(node.type === "section" || node.type === "container") && (
				<div style={S.propSection}>
					<div style={S.propSectionTitle}>Container</div>
					<Field label="Label"><Input value={p.label} onChange={v => onUpdateProps({ label: v })} /></Field>
					<Field label="Background"><ColorInput value={p.bg} onChange={v => onUpdateProps({ bg: v })} /></Field>
					<Field label="Padding"><NumberInput value={p.padding} onChange={v => onUpdateProps({ padding: v })} min={0} max={80} /></Field>
					<Field label="Border Radius"><NumberInput value={p.radius} onChange={v => onUpdateProps({ radius: v })} min={0} max={40} /></Field>
				</div>
			)}

			{node.type === "image" && (
				<div style={S.propSection}>
					<div style={S.propSectionTitle}>Image</div>
					<Field label="URL"><Input value={p.src} onChange={v => onUpdateProps({ src: v })} placeholder="https://..." /></Field>
					<Field label="Alt text"><Input value={p.alt} onChange={v => onUpdateProps({ alt: v })} /></Field>
					<Field label="Object Fit">
						<select value={p.objectFit || "cover"} onChange={e => onUpdateProps({ objectFit: e.target.value })} style={S.propInput}>
							{["cover", "contain", "fill", "none"].map(v => <option key={v}>{v}</option>)}
						</select>
					</Field>
					<Field label="Border Radius"><NumberInput value={p.radius} onChange={v => onUpdateProps({ radius: v })} min={0} max={40} /></Field>
				</div>
			)}

			{node.type === "divider" && (
				<div style={S.propSection}>
					<div style={S.propSectionTitle}>Divider</div>
					<Field label="Color"><ColorInput value={p.color} onChange={v => onUpdateProps({ color: v })} /></Field>
					<Field label="Thickness"><NumberInput value={p.thickness} onChange={v => onUpdateProps({ thickness: v })} min={1} max={20} /></Field>
				</div>
			)}
		</div>
	);
}

// ─── Layer Item ───────────────────────────────────────────────────────────────
function LayerItem({ id, nodes, selected, onSelect, depth }) {
	const node = nodes[id];
	if (!node) return null;
	const def = COMPONENT_DEFS[node.type];
	return (
		<div>
			<div
				style={{ ...S.layerItem, paddingLeft: 12 + depth * 14, background: selected === id ? "#f0f7ff" : "transparent", color: selected === id ? "#2563eb" : "#374151", borderLeft: selected === id ? "2px solid #6366f1" : "2px solid transparent" }}
				onClick={() => onSelect(id)}
			>
				<span style={{ color: def.color, marginRight: 6, fontSize: 11 }}>{def.icon}</span>
				<span style={{ fontSize: 12, flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{node.props?.label || def.label}</span>
			</div>
			{(node.children || []).map(cid => (
				<LayerItem key={cid} id={cid} nodes={nodes} selected={selected} onSelect={onSelect} depth={depth + 1} />
			))}
		</div>
	);
}

// ─── Styles ───────────────────────────────────────────────────────────────────
const S = {
	root: { fontFamily: "'DM Sans', sans-serif", height: "100vh", display: "flex", flexDirection: "column", background: "#f8fafc", overflow: "hidden" },
	topbar: { height: 48, background: "#111827", display: "flex", alignItems: "center", padding: "0 16px", gap: 12, flexShrink: 0, borderBottom: "1px solid #1f2937" },
	topbarLeft: { display: "flex", alignItems: "center", gap: 8, flex: 1 },
	brand: { fontFamily: "'Syne', sans-serif", fontWeight: 800, fontSize: 16, color: "#6366f1", letterSpacing: "-0.02em" },
	sep: { color: "#374151", fontSize: 14 },
	docName: { color: "#9ca3af", fontSize: 13 },
	topbarCenter: { display: "flex", gap: 4, background: "#1f2937", padding: "3px 4px", borderRadius: 6 },
	topbarRight: { display: "flex", gap: 8, flex: 1, justifyContent: "flex-end" },
	zoomBtn: { background: "transparent", border: "none", color: "#9ca3af", fontSize: 12, cursor: "pointer", padding: "3px 8px", borderRadius: 4, fontFamily: "'DM Sans', sans-serif" },
	zoomBtnActive: { background: "#374151", color: "#f9fafb" },
	topBtn: { padding: "5px 14px", background: "#1f2937", border: "1px solid #374151", color: "#d1d5db", borderRadius: 6, cursor: "pointer", fontSize: 12, fontFamily: "'DM Sans', sans-serif" },
	topBtnActive: { background: "#374151", color: "#fff" },
	topBtnPrimary: { background: "#6366f1", border: "none", color: "#fff" },
	workspace: { flex: 1, display: "flex", overflow: "hidden", minHeight: 0 },
	leftPanel: { width: 220, background: "#fff", borderRight: "1px solid #e5e7eb", overflowY: "auto", flexShrink: 0, padding: "0 0 16px" },
	panelHeader: { padding: "12px 14px 4px", fontSize: 11, fontWeight: 600, color: "#6b7280", letterSpacing: "0.06em", textTransform: "uppercase" },
	catLabel: { padding: "10px 14px 4px", fontSize: 10, fontWeight: 600, color: "#9ca3af", letterSpacing: "0.08em", textTransform: "uppercase" },
	paletteGrid: { display: "grid", gridTemplateColumns: "1fr 1fr", gap: 4, padding: "0 8px 4px" },
	paletteItem: { display: "flex", flexDirection: "column", alignItems: "center", gap: 3, padding: "8px 4px", background: "#f9fafb", border: "1px solid #e5e7eb", borderRadius: 6, cursor: "grab", transition: "border-color 0.15s, background 0.15s", userSelect: "none" },
	paletteIcon: { fontSize: 16, lineHeight: 1 },
	paletteLabel: { fontSize: 10, color: "#4b5563", textAlign: "center", lineHeight: 1.2 },
	layerItem: { display: "flex", alignItems: "center", padding: "6px 12px", cursor: "pointer", transition: "background 0.1s", fontSize: 12 },
	canvasWrap: { flex: 1, overflow: "hidden", position: "relative" },
	canvasScroll: { width: "100%", height: "100%", overflow: "auto", padding: 32, boxSizing: "border-box" },
	canvas: { display: "inline-block" },
	page: { width: 780, minHeight: 600, background: "#fff", borderRadius: 12, boxShadow: "0 4px 24px rgba(0,0,0,0.08)", position: "relative", overflow: "hidden" },
	emptyState: { position: "absolute", top: "50%", left: "50%", transform: "translate(-50%,-50%)", textAlign: "center", color: "#9ca3af", pointerEvents: "none" },
	emptyIcon: { fontSize: 40, marginBottom: 12, color: "#d1d5db" },
	emptyTitle: { fontFamily: "'Syne', sans-serif", fontWeight: 700, fontSize: 16, color: "#6b7280", marginBottom: 6 },
	emptyText: { fontSize: 13 },
	dropHint: { position: "absolute", top: "50%", left: "50%", transform: "translate(-50%,-50%)", fontSize: 11, color: "#d1d5db", pointerEvents: "none", textAlign: "center", letterSpacing: "0.04em" },
	dragHandle: { position: "absolute", top: -20, left: 0, height: 18, minWidth: 60, borderRadius: "4px 4px 0 0", cursor: "grab", zIndex: 200, display: "flex", alignItems: "center", paddingLeft: 6, paddingRight: 6, transition: "background 0.1s" },
	dragHandleLabel: { fontSize: 10, color: "#fff", whiteSpace: "nowrap", fontFamily: "'DM Sans', sans-serif", letterSpacing: "0.03em" },
	rightPanel: { width: 260, background: "#fff", borderLeft: "1px solid #e5e7eb", overflowY: "auto", flexShrink: 0 },
	propHeader: { display: "flex", alignItems: "center", justifyContent: "space-between", padding: "12px 14px", borderBottom: "1px solid #f1f5f9" },
	propTypeTag: { fontSize: 11, fontWeight: 600, padding: "3px 8px", borderRadius: 4, letterSpacing: "0.04em" },
	deleteBtn: { width: 26, height: 26, background: "#fef2f2", border: "1px solid #fecaca", color: "#ef4444", borderRadius: 4, cursor: "pointer", fontSize: 12, display: "flex", alignItems: "center", justifyContent: "center" },
	propBody: { padding: "8px 0" },
	propSection: { padding: "8px 14px 12px", borderBottom: "1px solid #f8fafc" },
	propSectionTitle: { fontSize: 10, fontWeight: 600, color: "#9ca3af", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 10 },
	field: { marginBottom: 10 },
	fieldLabel: { fontSize: 11, color: "#6b7280", marginBottom: 4, fontWeight: 500 },
	propInput: { width: "100%", padding: "6px 8px", border: "1px solid #e5e7eb", borderRadius: 5, fontSize: 12, fontFamily: "'DM Sans', sans-serif", color: "#111827", background: "#fafafa", boxSizing: "border-box", outline: "none" },
	alignBtn: { flex: 1, padding: "4px 8px", background: "#f3f4f6", border: "1px solid #e5e7eb", borderRadius: 4, fontSize: 11, cursor: "pointer", fontFamily: "'DM Sans', sans-serif", color: "#4b5563" },
	alignBtnActive: { background: "#6366f1", border: "1px solid #6366f1", color: "#fff" },
	formLabel: { fontSize: 12, fontWeight: 500, color: "#374151", fontFamily: "'DM Sans', sans-serif", display: "block" },
	formInput: { width: "100%", padding: "8px 10px", border: "1.5px solid #e5e7eb", borderRadius: 6, fontSize: 13, fontFamily: "'DM Sans', sans-serif", color: "#111827", background: "#fff", boxSizing: "border-box", outline: "none" },
	noSelection: { display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", height: 200, gap: 8, color: "#9ca3af", fontSize: 12, textAlign: "center" },
	noSelIcon: { fontSize: 24, color: "#d1d5db" },
};

const CSS = `
  * { box-sizing: border-box; }
  ::-webkit-scrollbar { width: 6px; height: 6px; }
  ::-webkit-scrollbar-track { background: transparent; }
  ::-webkit-scrollbar-thumb { background: #e5e7eb; border-radius: 3px; }
  [draggable]:active { cursor: grabbing; }
  button:hover { opacity: 0.88; }
  input:focus, textarea:focus, select:focus { border-color: #6366f1 !important; box-shadow: 0 0 0 3px #6366f120; }
`;
