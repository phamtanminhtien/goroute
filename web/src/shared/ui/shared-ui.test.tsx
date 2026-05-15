import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

const motionState = vi.hoisted(() => ({
  reducedMotion: false,
}));

vi.mock("motion/react", async () => {
  const actual =
    await vi.importActual<typeof import("motion/react")>("motion/react");

  return {
    ...actual,
    useReducedMotion: () => motionState.reducedMotion,
  };
});

import {
  AlertDialog,
  AlertDialogActionButton,
  AlertDialogCancelButton,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/shared/ui/alert-dialog";
import { Button } from "@/shared/ui/button";
import { Checkbox } from "@/shared/ui/checkbox";
import { Combobox } from "@/shared/ui/combobox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { InlineAlert } from "@/shared/ui/inline-alert";
import { Input } from "@/shared/ui/input";
import {
  Modal,
  ModalContent,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
  ModalTrigger,
} from "@/shared/ui/modal";
import { MultiSelect } from "@/shared/ui/multiselect";
import { PageHeader } from "@/shared/ui/page-header";
import { Progress } from "@/shared/ui/progress";
import { RadioGroup, RadioGroupItem } from "@/shared/ui/radio-group";
import { SectionCard } from "@/shared/ui/section-card";
import { Select } from "@/shared/ui/select";
import { Skeleton } from "@/shared/ui/skeleton";
import { Switch } from "@/shared/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";
import { Textarea } from "@/shared/ui/textarea";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/shared/ui/tooltip";
import { renderWithQueryClient } from "@/test/test-utils";

describe("shared ui primitives", () => {
  beforeEach(() => {
    motionState.reducedMotion = false;
  });

  it("renders field, page header, and section card in different contexts", () => {
    renderWithQueryClient(
      <div className="grid gap-4 md:grid-cols-2">
        <PageHeader
          description="Shared page description"
          eyebrow="Preview"
          title="Reusable title"
        />
        <SectionCard
          description="Shared section description"
          title="Section title"
          tone="solid"
        >
          <Field help="Helpful copy" label="API token" spacing="sm">
            <Input inputSize="lg" value="value" readOnly />
          </Field>
          <Button tone="secondary" size="lg" type="button">
            Save
          </Button>
        </SectionCard>
      </div>,
    );

    expect(screen.getByText(/reusable title/i)).toBeInTheDocument();
    expect(screen.getByText(/section title/i)).toBeInTheDocument();
    expect(screen.getByText(/api token/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
  });

  it("supports icon-only buttons with accessible names", () => {
    renderWithQueryClient(
      <Button
        aria-label="Reveal token"
        iconOnly
        leadingIcon={<span>+</span>}
      />,
    );

    expect(
      screen.getByRole("button", { name: /reveal token/i }),
    ).toBeInTheDocument();
  });

  it("creates a ripple on pointer down when enabled", () => {
    const { container } = renderWithQueryClient(
      <Button type="button">Create ripple</Button>,
    );

    const button = screen.getByRole("button", { name: /create ripple/i });
    fireEvent.pointerDown(button, { clientX: 16, clientY: 16 });

    expect(container.querySelector('span[aria-hidden="true"]')).not.toBeNull();
  });

  it("does not create a ripple when disabled", () => {
    const { container } = renderWithQueryClient(
      <Button disabled type="button">
        Disabled ripple
      </Button>,
    );

    fireEvent.pointerDown(
      screen.getByRole("button", { name: /disabled ripple/i }),
      {
        clientX: 16,
        clientY: 16,
      },
    );

    expect(container.querySelector('span[aria-hidden="true"]')).toBeNull();
  });

  it("does not create a ripple when reduced motion is preferred", () => {
    motionState.reducedMotion = true;

    const { container } = renderWithQueryClient(
      <Button type="button">Reduced motion</Button>,
    );

    fireEvent.pointerDown(
      screen.getByRole("button", { name: /reduced motion/i }),
      {
        clientX: 16,
        clientY: 16,
      },
    );

    expect(container.querySelector('span[aria-hidden="true"]')).toBeNull();
  });

  it("renders common shared controls and feedback primitives", () => {
    renderWithQueryClient(
      <div className="space-y-4">
        <Field help="Choose one or more options" label="Controls">
          <div className="space-y-3">
            <Checkbox aria-label="Enable logging" defaultChecked />
            <Switch aria-label="Enable sync" defaultChecked />
            <Textarea defaultValue="Longer notes" />
            <Select
              onValueChange={() => undefined}
              options={[
                { label: "Alpha", value: "alpha" },
                { label: "Beta", value: "beta" },
              ]}
              value="alpha"
            />
            <Progress value={64} />
          </div>
        </Field>
        <InlineAlert tone="info">Shared info state</InlineAlert>
        <EmptyState body="No results yet." title="Nothing here" />
        <Skeleton className="h-12" />
      </div>,
    );

    expect(
      screen.getByRole("checkbox", { name: /enable logging/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("switch", { name: /enable sync/i }),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue(/longer notes/i)).toBeInTheDocument();
    expect(screen.getByText(/shared info state/i)).toBeInTheDocument();
    expect(screen.getByText(/nothing here/i)).toBeInTheDocument();
  });

  it("supports keyboard-friendly alert dialogs and returns focus to the trigger", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button>Delete provider</Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete provider?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancelButton>Cancel</AlertDialogCancelButton>
            <AlertDialogActionButton>Confirm</AlertDialogActionButton>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>,
    );

    const trigger = screen.getByRole("button", { name: /delete provider/i });
    await user.click(trigger);

    const dialog = await screen.findByRole("alertdialog");
    expect(
      within(dialog).getByText(/this action cannot be undone/i),
    ).toBeInTheDocument();

    await user.keyboard("{Escape}");

    await waitFor(() => {
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    });
    expect(trigger).toHaveFocus();
  });

  it("opens modal and dropdown overlays", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <div className="flex gap-3">
        <Modal>
          <ModalTrigger asChild>
            <Button>Open modal</Button>
          </ModalTrigger>
          <ModalContent>
            <ModalHeader>
              <ModalTitle>Connection form</ModalTitle>
              <ModalDescription>Manage provider credentials.</ModalDescription>
            </ModalHeader>
            <ModalFooter>
              <Button tone="secondary">Close</Button>
            </ModalFooter>
          </ModalContent>
        </Modal>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button>Open menu</Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Rename</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>,
    );

    await user.click(screen.getByRole("button", { name: /open modal/i }));
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /close dialog/i }));
    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /open menu/i }));
    expect(await screen.findByRole("menu")).toBeInTheDocument();
  });

  it("supports searchable single and multi selection", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <div className="space-y-4">
        <Combobox
          options={[
            { keywords: ["first"], label: "Alpha", value: "alpha" },
            { keywords: ["second"], label: "Beta", value: "beta" },
          ]}
          value="alpha"
        />
        <MultiSelect
          defaultValue={["orchid"]}
          options={[
            { label: "Orchid", value: "orchid" },
            { label: "Moss", value: "moss" },
            { label: "Clay", value: "clay" },
          ]}
        />
      </div>,
    );

    await user.click(screen.getByRole("button", { name: /alpha/i }));
    await user.type(
      screen.getByRole("textbox", { name: /search options/i }),
      "be",
    );
    expect(screen.getByRole("option", { name: /beta/i })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /orchid/i }));
    await user.click(screen.getByRole("option", { name: /moss/i }));
    expect(
      screen.getByRole("button", { name: /orchidmoss/i }),
    ).toBeInTheDocument();
  });

  it("renders radio groups, tabs, and tooltips accessibly", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <TooltipProvider>
        <div className="space-y-4">
          <RadioGroup defaultValue="manual">
            <RadioGroupItem
              description="Explicit routing"
              label="Manual"
              value="manual"
            />
            <RadioGroupItem
              description="Automatic routing"
              label="Auto"
              value="auto"
            />
          </RadioGroup>
          <Tabs defaultValue="overview">
            <TabsList>
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="security">Security</TabsTrigger>
            </TabsList>
            <TabsContent value="overview">Overview tab</TabsContent>
            <TabsContent value="security">Security tab</TabsContent>
          </Tabs>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button>Hover target</Button>
            </TooltipTrigger>
            <TooltipContent>Helpful copy</TooltipContent>
          </Tooltip>
        </div>
      </TooltipProvider>,
    );

    expect(screen.getByRole("radio", { name: /manual/i })).toBeChecked();
    await user.click(screen.getByRole("tab", { name: /security/i }));
    expect(screen.getByText(/security tab/i)).toBeInTheDocument();

    await user.hover(screen.getByRole("button", { name: /hover target/i }));
    expect(await screen.findByRole("tooltip")).toHaveTextContent(
      /helpful copy/i,
    );
  });
});
