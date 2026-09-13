<script lang="ts">
  import ConfirmDialog from './ConfirmDialog.svelte'
  import { confirmRequest, answerConfirm } from '../confirm'
</script>

<ConfirmDialog
  open={$confirmRequest !== null}
  title={$confirmRequest?.title ?? ''}
  message={$confirmRequest?.message ?? ''}
  confirmLabel={$confirmRequest?.confirmLabel ?? ''}
  cancelLabel={$confirmRequest?.cancelLabel ?? ''}
  onConfirm={() => answerConfirm(true)}
  onCancel={() => answerConfirm(false)}
/>
