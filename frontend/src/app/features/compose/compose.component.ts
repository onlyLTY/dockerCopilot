import { Component, inject, signal, computed, DestroyRef } from "@angular/core";
import { takeUntilDestroyed } from "@angular/core/rxjs-interop";
import { FormsModule } from "@angular/forms";
import {
  ComposeService,
  ComposeProject,
  ProjectsData,
} from "../../core/compose.service";
import { ToastService } from "../../core/toast.service";
import { TaskService } from "../../core/task.service";
import { ConfirmService } from "../../core/confirm.service";
import { PageStateComponent } from "../../shared/page-state/page-state.component";
import { IconComponent } from "../../shared/icon/icon.component";
import { IconService } from "../../core/icon.service";
import { ResourceCardComponent } from "../../shared/resource-card/resource-card.component";
import { StatsComponent, StatItem } from "../../shared/stats/stats.component";
import { PageHeadingComponent } from "../../shared/page-heading/page-heading.component";
import { ModalHeadingComponent } from "../../shared/modal-heading/modal-heading.component";
import { actionErrorMessage, runAction } from "../../core/run-action";
import { MatTooltipModule } from "@angular/material/tooltip";

interface ComposeRisk {
  level?: string;
  field?: string;
  message?: string;
  kind?: string;
}

const defaultCompose = "";

@Component({
  selector: "dc-compose",
  standalone: true,
  imports: [
    FormsModule,
    PageStateComponent,
    IconComponent,
    ResourceCardComponent,
    StatsComponent,
    PageHeadingComponent,
    ModalHeadingComponent,
    MatTooltipModule,
  ],
  templateUrl: "./compose.component.html",
  styleUrl: "./compose.component.scss",
})
export class ComposeComponent {
  private readonly service = inject(ComposeService);
  readonly cacheStale = this.service.cache.stale;
  private readonly icons = inject(IconService);
  private readonly toast = inject(ToastService);
  private readonly tasks = inject(TaskService);
  private readonly confirm = inject(ConfirmService);
  private readonly destroyRef = inject(DestroyRef);
  // 行号：基于编辑器内容行数生成（模板 @for 渲染，避免 innerHTML 丢失 encapsulation）
  // 外层 textarea 滚动时同步 gutter 的 scrollTop
  readonly lineNumbers = computed(() => this.buildLineNumbers(this.content()));
  readonly createLineNumbers = computed(() =>
    this.buildLineNumbers(this.projectContent()),
  );
  private buildLineNumbers(text: string | undefined): number[] {
    const n = Math.max(1, (text || "").split("\n").length);
    return Array.from({ length: n }, (_, i) => i + 1);
  }
  onEditorScroll(e: Event) {
    this.syncGutter(e);
  }
  onCreateScroll(e: Event) {
    this.syncGutter(e);
  }
  private syncGutter(e: Event) {
    const ta = e.target as HTMLTextAreaElement;
    const gutter = ta.parentElement?.querySelector(
      ".code-gutter",
    ) as HTMLElement | null;
    if (gutter) gutter.scrollTop = ta.scrollTop;
  }
  readonly data = computed<ProjectsData | undefined>(() => {
    const value = this.service.cache.data();
    if (!value) return undefined;
    return {
      summary: value.summary || {
        total: 0,
        using: 0,
        stopped: 0,
        unused: 0,
        unknown: 0,
      },
      projects: Array.isArray(value.projects)
        ? value.projects.map((p) => ({
            ...p,
            files: Array.isArray(p.files) ? p.files : [],
            containers: Array.isArray(p.containers) ? p.containers : [],
            ports: Array.isArray(p.ports) ? p.ports : [],
          }))
        : [],
    };
  });
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly iconMap = computed(() => this.icons.cache.data() || {});
  readonly filter = signal("all");
  readonly filteredProjects = computed(() =>
    (this.data()?.projects || []).filter(
      (p) => this.filter() === "all" || p.status === this.filter(),
    ),
  );
  readonly stats = computed<readonly StatItem[]>(() => {
    const summary = this.data()?.summary;
    return [
      { key: "all", value: summary?.total || 0, label: "项目总数" },
      {
        key: "using",
        value: summary?.using || 0,
        label: "使用中",
        tone: "green",
      },
      {
        key: "stopped",
        value: summary?.stopped || 0,
        label: "已停止",
        tone: "amber",
      },
      {
        key: "unused",
        value: summary?.unused || 0,
        label: "未使用",
        tone: "violet",
      },
      {
        key: "unknown",
        value: summary?.unknown || 0,
        label: "未知",
        tone: "red",
      },
    ];
  });
  readonly selectionMode = signal(false);
  readonly selected = signal<Set<string>>(new Set());
  private pendingComposeTasks = new Set<string>();
  readonly cleanupBusy = signal(false);
  readonly backupBusy = signal(false);
  readonly deletingProject = signal<string | undefined>(undefined);
  readonly backingUpProject = signal<string | undefined>(undefined);
  readonly unusedProjects = computed(() =>
    (this.data()?.projects || []).filter((p) => p.status === "unused"),
  );
  readonly editorProject = signal<ComposeProject | undefined>(undefined);
  readonly filename = signal("");
  readonly content = signal("");
  readonly version = signal("");
  readonly savedContent = signal("");
  readonly savedVersion = signal("");
  readonly editorDirty = computed(
    () =>
      this.content() !== this.savedContent() ||
      this.version() !== this.savedVersion(),
  );
  readonly fileLoading = signal(false);
  readonly fileError = signal("");
  readonly message = signal("");
  readonly messageType = signal("");
  readonly pullImages = signal(false);
  readonly preview = signal<Record<string, unknown> | undefined>(undefined);
  readonly risks = signal<ComposeRisk[]>([]);
  readonly showCreate = signal(false);
  readonly creating = signal(false);
  readonly projectName = signal("");
  readonly projectContent = signal(defaultCompose);
  readonly createError = signal("");
  readonly createErrorType = signal<"error" | "success" | "">("");
  readonly confirmDeploy = signal(false);
  readonly deployBusy = signal(false);
  readonly createPreview = signal<Record<string, any> | undefined>(undefined);
  readonly createValidated = signal(false);
  readonly composeDialog = computed(() =>
    this.editorProject() ? "edit" : this.showCreate() ? "create" : "closed",
  );
  constructor() {
    this.service.ensureLoaded();
    this.icons.ensureLoaded();
    this.tasks.completed.pipe(takeUntilDestroyed()).subscribe(({ taskID }) => {
      if (!this.pendingComposeTasks.delete(taskID)) return;
      if (this.pendingComposeTasks.size === 0) {
        this.backupBusy.set(false);
        this.cleanupBusy.set(false);
      }
    });
  }
  refresh() {
    this.service.refresh();
  }
  selectFilter(key: string): void {
    this.filter.set(this.filter() === key || key === "all" ? "all" : key);
    this.selected.set(new Set());
  }
  openEditor(p: ComposeProject) {
    const normalized = {
      ...p,
      files: Array.isArray(p.files) ? p.files : [],
      containers: Array.isArray(p.containers) ? p.containers : [],
      ports: Array.isArray(p.ports) ? p.ports : [],
    };
    this.editorProject.set(normalized);
    this.filename.set("");
    this.content.set("");
    this.version.set("");
    this.savedContent.set("");
    this.savedVersion.set("");
    this.fileLoading.set(false);
    this.fileError.set("");
    this.preview.set(undefined);
    this.pullImages.set(false);
    this.message.set("");
    this.messageType.set("");
    const f =
      normalized.files.find((x) => x.name === "compose.yaml") ||
      normalized.files[0];
    if (f) this.openFile(f.name);
    else this.fileError.set("未找到 Compose 文件");
  }
  openFile(n: string) {
    const ep = this.editorProject();
    if (!ep || this.fileLoading() || n === this.filename()) return;
    if (this.editorDirty()) {
      this.confirm
        .open({
          title: "切换文件",
          message: "当前文件有未保存修改，切换后这些修改会丢失，确定继续吗？",
          confirmText: "放弃修改",
          danger: true,
        })
        .then((ok) => {
          if (ok) this.loadFile(ep, n);
        });
      return;
    }
    this.loadFile(ep, n);
  }

  private loadFile(ep: ComposeProject, n: string): void {
    this.filename.set(n);
    this.content.set("");
    this.version.set("");
    this.savedContent.set("");
    this.savedVersion.set("");
    this.fileError.set("");
    this.fileLoading.set(true);
    this.service.file(ep.id, n).subscribe({
      next: (r) => {
        if (this.editorProject()?.id !== ep.id || this.filename() !== n) return;
        this.fileLoading.set(false);
        if (r.code === 200) {
          this.content.set(r.data.content);
          this.version.set(r.data.version);
          this.savedContent.set(r.data.content);
          this.savedVersion.set(r.data.version);
          this.message.set("");
        } else {
          this.fileError.set(r.msg || "读取文件失败");
        }
      },
      error: (e) => {
        if (this.editorProject()?.id !== ep.id || this.filename() !== n) return;
        this.fileLoading.set(false);
        this.fileError.set(e.error?.msg || "读取文件失败");
      },
    });
  }
  save() {
    const ep = this.editorProject();
    if (!ep) return;
    runAction({
      request: this.service.update(
        ep.id,
        this.filename(),
        this.content(),
        this.version(),
      ),
      onSuccess: (r) => {
        this.version.set((r.data as { version: string }).version);
        this.savedContent.set(this.content());
        this.savedVersion.set(this.version());
        this.message.set("已保存");
        this.messageType.set("");
      },
      onBizError: (r) => this.showError(r.msg, "保存文件"),
      onHttpError: (e) =>
        this.showError(actionErrorMessage(e, "保存失败"), "保存文件"),
    });
  }
  validate() {
    const ep = this.editorProject();
    if (!ep) return;
    this.content.set(this.normalizeCompose(this.content()));
    runAction({
      request: this.service.validate(ep.id, this.filename(), this.content()),
      onSuccess: (r) => {
        const n = Array.isArray(
          (r.data as { services?: unknown[] } | undefined)?.services,
        )
          ? (r.data as { services: unknown[] }).services.length
          : 0;
        this.message.set("校验通过，包含 " + n + " 个服务");
        this.messageType.set("");
      },
      onBizError: (r) => this.showError(r.msg, "校验"),
      onHttpError: (e) =>
        this.showError(actionErrorMessage(e, "校验失败"), "校验"),
    });
  }
  redeploy() {
    const ep = this.editorProject();
    if (!ep || this.deployBusy()) return;
    this.deployBusy.set(true);
    this.message.set("正在保存并校验配置…");
    this.service
      .update(ep.id, this.filename(), this.content(), this.version())
      .subscribe({
        next: (saved) => {
          if (saved.code !== 200) {
            this.deployBusy.set(false);
            this.showError(saved.msg, "重新部署");
            return;
          }
          this.version.set(saved.data.version);
          this.savedContent.set(this.content());
          this.savedVersion.set(this.version());
          this.service
            .validate(ep.id, this.filename(), this.content())
            .subscribe({
              next: (validated) => {
                if (validated.code !== 200) {
                  this.deployBusy.set(false);
                  this.showError(validated.msg, "重新部署");
                  return;
                }
                this.service.deployPreview(ep.id, this.filename()).subscribe({
                  next: (preview) => {
                    this.deployBusy.set(false);
                    if (preview.code !== 200) {
                      this.showError(preview.msg, "重新部署");
                      return;
                    }
                    this.askDeployment(
                      ep,
                      this.filename(),
                      preview.data,
                      this.pullImages(),
                    );
                  },
                  error: (e) => {
                    this.deployBusy.set(false);
                    this.showError(e.error?.msg || "部署检查失败", "重新部署");
                  },
                });
              },
              error: (e) => {
                this.deployBusy.set(false);
                this.showError(e.error?.msg || "格式校验失败", "重新部署");
              },
            });
        },
        error: (e) => {
          this.deployBusy.set(false);
          this.showError(e.error?.msg || "保存失败", "重新部署");
        },
      });
  }
  private askDeployment(
    project: ComposeProject,
    filename: string,
    preview: Record<string, any>,
    pullImages = false,
  ) {
    const risks = this.readRisks(preview);
    const critical = risks.filter((r) => this.isCriticalRisk(r));
    const riskDetails = risks
      .map(
        (risk, index) =>
          `${index + 1}. [${this.riskLevelLabel(risk)}] ${risk.field || "配置"}：${risk.message || "请检查此配置"}${risk.kind ? `（${risk.kind}）` : ""}`,
      )
      .join("\n");
    const riskMessage = risks.length
      ? `检测到 ${risks.length} 项风险${critical.length ? `，其中 ${critical.length} 项为极高危配置` : ""}，将鼠标悬停查看完整风险。`
      : "";
    this.confirm
      .open({
        title: "确认部署",
        message: `即将部署项目 ${project.name || project.id}。${riskMessage}${pullImages ? "将重新拉取项目镜像。" : ""}`,
        messageTooltip: riskDetails || undefined,
        confirmText: "确定部署",
        critical: critical.length > 0,
      })
      .then((ok) => {
        if (ok) this.confirmDeployment(project, filename, preview, pullImages);
      });
  }
  private readRisks(preview: Record<string, any>): ComposeRisk[] {
    return Array.isArray(preview["risks"])
      ? (preview["risks"] as ComposeRisk[])
      : [];
  }

  private isCriticalRisk(risk: ComposeRisk): boolean {
    return ["docker_socket", "privileged", "sensitive_host_path"].includes(
      risk.kind || "",
    );
  }

  private riskLevelLabel(risk: ComposeRisk): string {
    if (this.isCriticalRisk(risk)) return "极高危";
    return risk.level === "high" ? "高风险" : "提示";
  }

  private confirmDeployment(
    project: ComposeProject,
    filename: string,
    preview: Record<string, any>,
    pullImages = false,
  ) {
    if (this.deployBusy()) return;
    const risks = this.readRisks(preview);
    runAction({
      request: this.service.deploy(
        project.id,
        filename,
        String(preview["confirmToken"]),
        risks.length > 0,
        pullImages,
      ),
      onStart: () => this.deployBusy.set(true),
      onFinally: () => this.deployBusy.set(false),
      onSuccess: (r) => {
        const taskID = (r.data as { taskID?: string } | undefined)?.taskID;
        if (taskID) {
          this.tasks.track(
            String(taskID),
            "部署 " + (project.name || project.id),
            true,
          );
          this.closeEditor(true);
        } else {
          this.toast.error("部署失败：服务未返回任务编号");
        }
      },
      onBizError: (r) => this.showError(r.msg, "部署"),
      onHttpError: (e) =>
        this.showError(actionErrorMessage(e, "部署失败"), "部署"),
    });
  }
  newProject() {
    this.projectName.set("");
    this.projectContent.set(defaultCompose);
    this.createError.set("");
    this.createValidated.set(false);
    this.showCreate.set(true);
  }
  validateCreate() {
    const n = this.projectName().trim();
    if (!n) {
      this.setCreateError("请输入项目名称", "error");
      return;
    }
    this.projectContent.set(this.normalizeCompose(this.projectContent()));
    runAction({
      request: this.service.validate("", "compose.yaml", this.projectContent()),
      onSuccess: (r) => {
        const count = Array.isArray(
          (r.data as { services?: unknown[] } | undefined)?.services,
        )
          ? (r.data as { services: unknown[] }).services.length
          : 0;
        this.createValidated.set(true);
        this.setCreateError(
          "格式校验通过，包含 " + count + " 个服务",
          "success",
        );
      },
      onBizError: (r) =>
        this.setCreateError(this.formatComposeError(r.msg), "error"),
      onHttpError: (e) =>
        this.setCreateError(
          this.formatComposeError(actionErrorMessage(e, "格式校验失败")),
          "error",
        ),
    });
  }
  deployNewProject() {
    this.createProject(true);
  }
  createProject(deployAfterCreate = false) {
    const n = this.projectName().trim();
    if (!n) {
      this.setCreateError("请输入项目名称", "error");
      return;
    }
    if (deployAfterCreate && !this.createValidated()) {
      this.setCreateError("创建并部署前请先完成格式校验", "error");
      return;
    }
    this.projectContent.set(this.normalizeCompose(this.projectContent()));
    runAction({
      request: this.service.createProject(
        n,
        "compose.yaml",
        this.projectContent(),
      ),
      onStart: () => this.creating.set(true),
      onFinally: () => this.creating.set(false),
      onSuccess: (r) => {
        if (!deployAfterCreate) {
          this.closeCreate();
          this.toast.success("项目已创建");
          return;
        }
        this.createAndPreview((r.data as { projectId: string }).projectId);
      },
      onBizError: (r) =>
        this.setCreateError(
          this.formatComposeError(r.msg || "创建项目失败"),
          "error",
        ),
      onHttpError: (e) =>
        this.setCreateError(
          this.formatComposeError(actionErrorMessage(e, "创建项目失败")),
          "error",
        ),
    });
  }
  private createAndPreview(projectId: string) {
    const filename = "compose.yaml";
    this.service.deployPreview(projectId, filename).subscribe({
      next: (preview) => {
        if (preview.code !== 200) {
          this.setCreateError(
            this.formatComposeError(preview.msg || "部署预览失败"),
            "error",
          );
          return;
        }
        this.closeCreate();
        const project: ComposeProject = {
          id: projectId,
          name: String(preview.data?.["projectId"] ?? projectId),
          image: "",
          root: "",
          files: [{ name: filename, size: 0, modifiedAt: "", valid: true }],
          containers: [],
          ports: [],
          status: "unused",
        };
        this.askDeployment(project, filename, preview.data);
      },
      error: (e) =>
        this.setCreateError(
          this.formatComposeError(e.error?.msg || "部署预览失败"),
          "error",
        ),
    });
  }

  async backupProject(project: ComposeProject): Promise<void> {
    if (this.backingUpProject() || this.cleanupBusy() || this.deletingProject())
      return;
    this.backingUpProject.set(project.id);
    this.service.backup(project.id).subscribe({
      next: (result) => {
        this.backingUpProject.set(undefined);
        if (result.code === 200) {
          const filename = String(result.data?.["filename"] || "compose.yaml");
          this.toast.success(
            `项目 ${project.name || project.id} 已备份为 ${filename}`,
          );
          return;
        }
        this.toast.error(`备份项目失败：${result.msg || "未知错误"}`);
      },
      error: (error) => {
        this.backingUpProject.set(undefined);
        this.toast.error(`备份项目失败：${actionErrorMessage(error)}`);
      },
    });
  }
  async backupAllProjects(): Promise<void> {
    const projects = this.data()?.projects || [];
    if (
      !projects.length ||
      this.backupBusy() ||
      this.cleanupBusy() ||
      this.deletingProject() ||
      this.backingUpProject()
    )
      return;
    this.backupBusy.set(true);
    this.service.backupBatch(projects.map((project) => project.id)).subscribe({
      next: (result) => {
        if (result.code !== 200) {
          this.backupBusy.set(false);
          this.toast.error(`批量备份失败：${result.msg || "无法创建任务"}`);
          return;
        }
        const taskID = result.data?.taskID;
        if (!taskID) {
          this.backupBusy.set(false);
          this.toast.error("批量备份失败：服务未返回任务编号");
          return;
        }
        this.pendingComposeTasks.add(String(taskID));
        this.tasks.track(String(taskID), "批量备份 Compose 项目", true);
        this.toast.success("批量备份任务已提交，可在任务页查看详细结果");
      },
      error: (error) => {
        this.backupBusy.set(false);
        this.toast.error(`批量备份失败：${actionErrorMessage(error)}`);
      },
    });
  }

  async deleteProject(project: ComposeProject): Promise<void> {
    if (
      project.status !== "unused" ||
      this.cleanupBusy() ||
      this.deletingProject() ||
      this.backupBusy()
    )
      return;
    const label = project.name || project.id;
    if (
      !(await this.confirm.open({
        title: "删除项目文件夹",
        message: `确定删除项目 ${label} 吗？将递归删除整个项目文件夹及其中的所有文件和子目录，此操作不可恢复。`,
        confirmText: "确认删除",
        danger: true,
        critical: true,
      }))
    )
      return;

    this.deletingProject.set(project.id);
    this.service.cleanupPreview(project.id).subscribe({
      next: (preview) => {
        const token = preview.data?.["previewToken"];
        if (preview.code !== 200 || !token) {
          this.deletingProject.set(undefined);
          this.toast.error(
            `删除项目失败：${preview.msg || "无法生成删除预览"}`,
          );
          return;
        }
        this.service.cleanup(project.id, String(token), true).subscribe({
          next: (result) => {
            this.deletingProject.set(undefined);
            if (result.code === 200) {
              this.toast.success(`项目 ${label} 已删除`);
              this.service.refresh();
              return;
            }
            this.toast.error(`删除项目失败：${result.msg || "未知错误"}`);
          },
          error: (error) => {
            this.deletingProject.set(undefined);
            this.toast.error(`删除项目失败：${actionErrorMessage(error)}`);
          },
        });
      },
      error: (error) => {
        this.deletingProject.set(undefined);
        this.toast.error(
          `删除项目失败：${actionErrorMessage(error, "无法生成删除预览")}`,
        );
      },
    });
  }

  enterSelection() {
    this.selectionMode.set(true);
  }
  exitSelection() {
    this.selectionMode.set(false);
    this.selected.set(new Set());
  }
  isSelected(id: string) {
    return this.selected().has(id);
  }
  toggleSelect(id: string) {
    const next = new Set(this.selected());
    next.has(id) ? next.delete(id) : next.add(id);
    this.selected.set(next);
  }
  readonly allSelected = computed(
    () =>
      (this.data()?.projects || []).length > 0 &&
      (this.data()?.projects || []).every((p) => this.selected().has(p.id)),
  );
  toggleAll() {
    const projects = this.data()?.projects || [];
    const next = new Set(this.selected());
    if (this.allSelected()) projects.forEach((p) => next.delete(p.id));
    else projects.forEach((p) => next.add(p.id));
    this.selected.set(next);
  }
  deleteUnusedProjects() {
    const projects = this.unusedProjects();
    if (
      !projects.length ||
      this.cleanupBusy() ||
      this.deletingProject() ||
      this.backupBusy()
    )
      return;
    this.confirm
      .open({
        title: "删除未使用项目",
        message: `将逐个递归删除 ${projects.length} 个未使用项目的整个文件夹及其中所有数据，此操作不可恢复。确定继续吗？`,
        confirmText: "确认删除",
        danger: true,
        critical: true,
      })
      .then((ok) => {
        if (ok) this.runBulkCleanup(projects);
      });
  }

  bulkCleanup() {
    const projects = (this.data()?.projects || []).filter(
      (p) => this.selected().has(p.id) && p.status === "unused",
    );
    if (!projects.length || this.cleanupBusy() || this.backupBusy()) return;
    this.confirm
      .open({
        title: "清理未使用项目",
        message: `确定清理选中的 ${projects.length} 个未使用项目吗？`,
        confirmText: "确认清理",
        danger: true,
      })
      .then((ok) => {
        if (ok) this.runBulkCleanup(projects);
      });
  }
  private runBulkCleanup(projects: ComposeProject[]) {
    this.cleanupBusy.set(true);
    this.service
      .cleanupBatch(
        projects.map((project) => project.id),
        true,
      )
      .subscribe({
        next: (result) => {
          if (result.code !== 200) {
            this.toast.error(`项目删除失败：${result.msg || "无法创建任务"}`);
            return;
          }
          const taskID = result.data?.taskID;
          if (!taskID) {
            this.cleanupBusy.set(false);
            this.exitSelection();
            this.toast.error("项目删除失败：服务未返回任务编号");
            return;
          }
          this.exitSelection();
          this.pendingComposeTasks.add(String(taskID));
          this.tasks.track(String(taskID), "批量删除 Compose 项目", true);
          this.toast.info(`已提交 ${projects.length} 个项目的删除任务`);
        },
        error: (error) => {
          this.cleanupBusy.set(false);
          this.exitSelection();
          this.toast.error(`项目删除失败：${actionErrorMessage(error)}`);
        },
      });
  }
  private finishCleanup(total: number, failed: number, errors: string[] = []) {
    this.cleanupBusy.set(false);
    this.exitSelection();
    this.service.refresh();
    if (failed) {
      const detail = errors.length ? `：${errors.join("；")}` : "";
      this.toast.error(
        `项目删除完成 ${total - failed} 个，失败 ${failed} 个${detail}`,
      );
    } else this.toast.success(`已删除 ${total} 个未使用项目`);
  }
  closeEditor(e?: Event | boolean) {
    if (e === true) {
      this.editorProject.set(undefined);
      return;
    }
    if (e && typeof e !== "boolean" && e.target !== e.currentTarget) return;
    if (this.deployBusy() || this.fileLoading()) return;
    if (!this.editorDirty()) {
      this.editorProject.set(undefined);
      return;
    }
    this.confirm
      .open({
        title: "放弃未保存修改",
        message:
          "当前 Compose 文件存在未保存修改，关闭后将丢失这些内容，确定关闭吗？",
        confirmText: "放弃修改",
        danger: true,
      })
      .then((ok) => {
        if (ok) this.editorProject.set(undefined);
      });
  }
  closeCreate(e?: Event) {
    if (!e || e.target === e.currentTarget) {
      this.showCreate.set(false);
      this.creating.set(false);
      this.createError.set("");
      this.createValidated.set(false);
      this.projectName.set("");
      this.projectContent.set(defaultCompose);
    }
  }
  private normalizeCompose(input: string): string {
    // 将行首/行内非 ASCII 空白（NBSP、全角空格、零宽字符等）归一化为 ASCII 空格，
    // 避免 yaml.v3 解析报 "could not find expected ':'"。
    if (!input) return input;
    return input
      .split("\n")
      .map((line) =>
        line
          .replace(/[\u00A0\u1680\u2000-\u200A\u202F\u205F\u3000\uFEFF]/g, " ")
          .replace(/[\u200B\u200C\u200D]/g, ""),
      )
      .join("\n");
  }
  private formatComposeError(detail?: string): string {
    if (!detail) return "未知错误";
    let msg = String(detail);
    const m = msg.match(/YAML 解析失败:\s*(.*)/);
    if (m && m[1]) msg = "YAML 解析失败：" + m[1].trim();
    const lm = msg.match(/line\s+(\d+)/i);
    if (lm) msg = msg.replace(/line\s+(\d+)/i, "第 $1 行");
    return msg;
  }
  private setCreateError(detail: string, type: "error" | "success"): void {
    if (!detail) {
      this.createError.set("");
      this.createErrorType.set("");
      return;
    }
    this.createError.set(detail);
    this.createErrorType.set(type);
  }
  private setEditorError(detail: string): void {
    if (!detail) {
      this.message.set("");
      this.messageType.set("");
      return;
    }
    this.message.set(detail);
    this.messageType.set("error");
  }
  private showError(detail: string, operation = "操作"): void {
    this.setEditorError(detail);
  }
  statusLabel(s: string) {
    return (
      (
        {
          using: "使用中",
          stopped: "已停止",
          unused: "未使用",
          unknown: "未知",
        } as Record<string, string>
      )[s] || s
    );
  }
  icon(project: ComposeProject) {
    return this.icons.resolve(project.image || "", this.iconMap());
  }
  fallback(event: Event) {
    (event.target as HTMLImageElement).src = this.icons.defaultIcon();
  }
}
