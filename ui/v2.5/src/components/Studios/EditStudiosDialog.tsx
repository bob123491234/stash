import React, { useEffect, useState } from "react";
import { Form } from "react-bootstrap";
import { FormattedMessage, useIntl } from "react-intl";
import { useBulkStudioUpdate } from "src/core/StashService";
import * as GQL from "src/core/generated-graphql";
import { StudioSelect } from "src/components/Shared/Select";
import { ModalComponent } from "../Shared/Modal";
import { useToast } from "src/hooks/Toast";
import { MultiSet } from "../Shared/MultiSet";
import {
  getAggregateStudioId,
  getAggregateState,
  getAggregateStateObject,
} from "src/utils/bulkUpdate";
import { IndeterminateCheckbox } from "../Shared/IndeterminateCheckbox";
import { BulkUpdateTextInput } from "../Shared/BulkUpdateTextInput";
import { faPencilAlt } from "@fortawesome/free-solid-svg-icons";

function Studios(props: {
  isUpdating: boolean;
  controlId: string;
  messageId: string;
  existingStudioIds: string[] | undefined;
  studioIDs: GQL.BulkUpdateIds;
  setStudioIDs: (value: React.SetStateAction<GQL.BulkUpdateIds>) => void;
}) {
  const {
    isUpdating,
    controlId,
    messageId,
    existingStudioIds,
    studioIDs,
    setStudioIDs,
  } = props;

  return (
    <Form.Group controlId={controlId}>
      <Form.Label>
        <FormattedMessage id={messageId} />
      </Form.Label>
      <MultiSet
        type="studios"
        disabled={isUpdating}
        onUpdate={(itemIDs) =>
          setStudioIDs((existing) => ({ ...existing, ids: itemIDs }))
        }
        onSetMode={(newMode) =>
          setStudioIDs((existing) => ({ ...existing, mode: newMode }))
        }
        existingIds={existingStudioIds ?? []}
        ids={studioIDs.ids ?? []}
        mode={studioIDs.mode}
      />
    </Form.Group>
  );
}

interface IListOperationProps {
  selected: GQL.SlimStudioDataFragment[];
  onClose: (applied: boolean) => void;
}

const studioFields = ["favorite", "details", "ignore_auto_tag"];

export const EditStudiosDialog: React.FC<IListOperationProps> = (
  props: IListOperationProps
) => {
  const intl = useIntl();
  const Toast = useToast();

  const [parentStudioID, setParentStudioID_] = useState<string>();

  function setParentStudioID(value: React.SetStateAction<GQL.BulkUpdateIds>) {
    console.log(value);
    setParentStudioID_(value);
  }

  const [existingParentStudioId, setExistingParentStudioId] = useState<string[]>();

  const [updateInput, setUpdateInput] = useState<GQL.BulkStudioUpdateInput>({});

  const [updateStudios] = useBulkStudioUpdate(getStudioInput());

  // Network state
  const [isUpdating, setIsUpdating] = useState(false);

  function setUpdateField(input: Partial<GQL.BulkStudioUpdateInput>) {
    setUpdateInput({ ...updateInput, ...input });
  }

  function getStudioInput(): GQL.BulkStudioUpdateInput {
    const aggregateParentStudioId = getAggregateStudioId(props.selected);
    
    const studioInput: GQL.BulkStudioUpdateInput = {
      ids: props.selected.map((studio) => {
        return studio.id;
      }),
      ...updateInput,
    };

    return studioInput;
  }

  async function onSave() {
    setIsUpdating(true);
    try {
      await updateStudios();
      Toast.success(
        intl.formatMessage(
          { id: "toast.updated_entity" },
          {
            entity: intl.formatMessage({ id: "studios" }).toLocaleLowerCase(),
          }
        )
      );
      props.onClose(true);
    } catch (e) {
      Toast.error(e);
    }
    setIsUpdating(false);
  }

  useEffect(() => {
    const updateState: GQL.BulkStudioUpdateInput = {};

    const state = props.selected;
    let updateParentStudioIds: string[] = [];
    let first = true;

    state.forEach((studio: GQL.StudioDataFragment) => {
      getAggregateStateObject(updateState, studio, studioFields, first);

      const thisParents = (studio.parents ?? []).map((t) => t.id).sort();
      updateParentStudioIds =
        getAggregateState(updateParentStudioIds, thisParents, first) ?? [];

      first = false;
    });

    setExistingParentStudioIds(updateParentStudioIds);
    setUpdateInput(updateState);
  }, [props.selected]);

  function renderTextField(
    name: string,
    value: string | undefined | null,
    setter: (newValue: string | undefined) => void
  ) {
    return (
      <Form.Group controlId={name}>
        <Form.Label>
          <FormattedMessage id={name} />
        </Form.Label>
        <BulkUpdateTextInput
          value={value === null ? "" : value ?? undefined}
          valueChanged={(newValue) => setter(newValue)}
          unsetDisabled={props.selected.length < 2}
        />
      </Form.Group>
    );
  }

  return (
    <ModalComponent
      dialogClassName="edit-studios-dialog"
      show
      icon={faPencilAlt}
      header={intl.formatMessage(
        { id: "actions.edit_entity" },
        { entityType: intl.formatMessage({ id: "studios" }) }
      )}
      accept={{
        onClick: onSave,
        text: intl.formatMessage({ id: "actions.apply" }),
      }}
      cancel={{
        onClick: () => props.onClose(false),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={isUpdating}
    >
      <Form>
        <Form.Group controlId="favorite">
          <IndeterminateCheckbox
            setChecked={(checked) => setUpdateField({ favorite: checked })}
            checked={updateInput.favorite ?? undefined}
            label={intl.formatMessage({ id: "favourite" })}
          />
        </Form.Group>

        {renderTextField("details", updateInput.details, (v) =>
          setUpdateField({ details: v })
        )}

        <Studios
          isUpdating={isUpdating}
          controlId="parent-studio"
          messageId="parent_studio"
          existingStudioIds={existingParentStudioId}
          studioIDs={parentStudioID}
          setStudioIDs={setParentStudioID}
        />

        <Form.Group controlId="ignore-auto-tags">
          <IndeterminateCheckbox
            label={intl.formatMessage({ id: "ignore_auto_tag" })}
            setChecked={(checked) =>
              setUpdateField({ ignore_auto_tag: checked })
            }
            checked={updateInput.ignore_auto_tag ?? undefined}
          />
        </Form.Group>
      </Form>
    </ModalComponent>
  );
};
