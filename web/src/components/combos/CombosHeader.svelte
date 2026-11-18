<script lang="ts">
  import Button from '../../lib/ui/Button.svelte'

  interface Props {
    onCreateClick: () => void
  }

  let { onCreateClick }: Props = $props()
</script>

<div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
  <div>
    <h1 class="text-2xl font-semibold">Combos</h1>
    <p class="text-sm text-text-muted mt-1">
      Group models under one name, then pick a strategy per combo:
      <span class="text-text-main">Fallback</span> tries in order,
      <span class="text-text-main">Round Robin</span> rotates, and
      <span class="text-text-main">Fusion</span> panels all models through a judge.
    </p>
  </div>
  <div class="shrink-0">
    <Button icon="add" size="sm" onclick={onCreateClick} class="whitespace-nowrap">
      Create Combo
    </Button>
  </div>
</div>
