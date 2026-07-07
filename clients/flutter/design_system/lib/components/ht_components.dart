import 'package:flutter/material.dart';
import '../tokens/ht_tokens.dart';

/// HelixTerminator Design System Component Widgets
///
/// Reusable widgets that map directly to the design system components.

// ── Buttons ──────────────────────────────────────────────

class HTPrimaryButton extends StatelessWidget {
  final String label;
  final VoidCallback? onPressed;
  final Widget? icon;
  final bool isLoading;
  final bool fullWidth;
  final ButtonSize size;

  const HTPrimaryButton({
    super.key,
    required this.label,
    this.onPressed,
    this.icon,
    this.isLoading = false,
    this.fullWidth = false,
    this.size = ButtonSize.medium,
  });

  @override
  Widget build(BuildContext context) {
    final style = ElevatedButton.styleFrom(
      backgroundColor: HTTokens.color.brandPrimary,
      foregroundColor: HTTokens.color.textInverse,
      minimumSize: Size(fullWidth ? double.infinity : 0, size.height),
      padding: EdgeInsets.symmetric(horizontal: size.horizontalPadding, vertical: size.verticalPadding),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
      ),
      textStyle: Theme.of(context).textTheme.labelLarge?.copyWith(
        fontWeight: HTTokens.typography.weightSemibold,
      ),
      elevation: 0,
    );

    final child = isLoading
        ? SizedBox(
            height: 20,
            width: 20,
            child: CircularProgressIndicator(
              strokeWidth: 2,
              color: HTTokens.color.textInverse,
            ),
          )
        : Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (icon != null) ...[
                icon!,
                const SizedBox(width: 8),
              ],
              Text(label),
            ],
          );

    return ElevatedButton(
      onPressed: isLoading ? null : onPressed,
      style: style,
      child: child,
    );
  }
}

class HTSecondaryButton extends StatelessWidget {
  final String label;
  final VoidCallback? onPressed;
  final Widget? icon;
  final bool fullWidth;
  final ButtonSize size;

  const HTSecondaryButton({
    super.key,
    required this.label,
    this.onPressed,
    this.icon,
    this.fullWidth = false,
    this.size = ButtonSize.medium,
  });

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return OutlinedButton(
      onPressed: onPressed,
      style: OutlinedButton.styleFrom(
        foregroundColor: isDark ? HTTokens.color.textPrimary : HTTokens.color.textPrimaryLight,
        minimumSize: Size(fullWidth ? double.infinity : 0, size.height),
        padding: EdgeInsets.symmetric(horizontal: size.horizontalPadding, vertical: size.verticalPadding),
        side: BorderSide(
          color: isDark ? HTTokens.color.borderDefault : HTTokens.color.borderDefaultLight,
          width: HTTokens.border.widthThin,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
        ),
        textStyle: Theme.of(context).textTheme.labelLarge,
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            icon!,
            const SizedBox(width: 8),
          ],
          Text(label),
        ],
      ),
    );
  }
}

class HTDangerButton extends StatelessWidget {
  final String label;
  final VoidCallback? onPressed;
  final bool fullWidth;
  final ButtonSize size;

  const HTDangerButton({
    super.key,
    required this.label,
    this.onPressed,
    this.fullWidth = false,
    this.size = ButtonSize.medium,
  });

  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: onPressed,
      style: ElevatedButton.styleFrom(
        backgroundColor: HTTokens.color.brandDanger,
        foregroundColor: HTTokens.color.textInverse,
        minimumSize: Size(fullWidth ? double.infinity : 0, size.height),
        padding: EdgeInsets.symmetric(horizontal: size.horizontalPadding, vertical: size.verticalPadding),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
        ),
        textStyle: Theme.of(context).textTheme.labelLarge?.copyWith(
          fontWeight: HTTokens.typography.weightSemibold,
        ),
        elevation: 0,
      ),
      child: Text(label),
    );
  }
}

class HTGhostButton extends StatelessWidget {
  final String? label;
  final Widget? icon;
  final VoidCallback? onPressed;

  const HTGhostButton({
    super.key,
    this.label,
    this.icon,
    this.onPressed,
  }) : assert(label != null || icon != null);

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return TextButton(
      onPressed: onPressed,
      style: TextButton.styleFrom(
        foregroundColor: isDark ? HTTokens.color.textSecondary : HTTokens.color.textSecondaryLight,
        minimumSize: const Size(44, 44),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) icon!,
          if (icon != null && label != null) const SizedBox(width: 8),
          if (label != null) Text(label!),
        ],
      ),
    );
  }
}

class HTIconButton extends StatelessWidget {
  final Widget icon;
  final VoidCallback? onPressed;
  final String tooltip;
  final bool isActive;

  const HTIconButton({
    super.key,
    required this.icon,
    this.onPressed,
    required this.tooltip,
    this.isActive = false,
  });

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return IconButton(
      onPressed: onPressed,
      icon: icon,
      tooltip: tooltip,
      style: IconButton.styleFrom(
        foregroundColor: isActive
            ? HTTokens.color.brandPrimary
            : isDark
                ? HTTokens.color.textSecondary
                : HTTokens.color.textSecondaryLight,
        minimumSize: const Size(44, 44),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusMd),
        ),
      ),
    );
  }
}

enum ButtonSize {
  small(height: 32, horizontalPadding: 12, verticalPadding: 8),
  medium(height: 40, horizontalPadding: 16, verticalPadding: 12),
  large(height: 48, horizontalPadding: 24, verticalPadding: 16);

  final double height;
  final double horizontalPadding;
  final double verticalPadding;

  const ButtonSize({
    required this.height,
    required this.horizontalPadding,
    required this.verticalPadding,
  });
}

// ── Inputs ─────────────────────────────────────────────

class HTTextInput extends StatelessWidget {
  final String? label;
  final String? hint;
  final String? helperText;
  final String? errorText;
  final TextEditingController? controller;
  final bool obscureText;
  final Widget? prefixIcon;
  final Widget? suffixIcon;
  final TextInputType? keyboardType;
  final void Function(String)? onChanged;
  final void Function(String)? onSubmitted;
  final int? maxLines;
  final bool enabled;
  final FocusNode? focusNode;

  const HTTextInput({
    super.key,
    this.label,
    this.hint,
    this.helperText,
    this.errorText,
    this.controller,
    this.obscureText = false,
    this.prefixIcon,
    this.suffixIcon,
    this.keyboardType,
    this.onChanged,
    this.onSubmitted,
    this.maxLines = 1,
    this.enabled = true,
    this.focusNode,
  });

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      obscureText: obscureText,
      keyboardType: keyboardType,
      onChanged: onChanged,
      onSubmitted: onSubmitted,
      maxLines: maxLines,
      enabled: enabled,
      focusNode: focusNode,
      style: Theme.of(context).textTheme.bodyMedium,
      decoration: InputDecoration(
        labelText: label,
        hintText: hint,
        helperText: helperText,
        errorText: errorText,
        prefixIcon: prefixIcon != null
            ? Padding(
                padding: const EdgeInsets.all(12),
                child: prefixIcon,
              )
            : null,
        suffixIcon: suffixIcon != null
            ? Padding(
                padding: const EdgeInsets.all(12),
                child: suffixIcon,
              )
            : null,
      ),
    );
  }
}

class HTSearchInput extends StatelessWidget {
  final String? hint;
  final TextEditingController? controller;
  final void Function(String)? onChanged;
  final VoidCallback? onClear;

  const HTSearchInput({
    super.key,
    this.hint = 'Search...',
    this.controller,
    this.onChanged,
    this.onClear,
  });

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return TextField(
      controller: controller,
      onChanged: onChanged,
      style: Theme.of(context).textTheme.bodyMedium,
      decoration: InputDecoration(
        filled: true,
        fillColor: isDark ? HTTokens.color.backgroundTertiary : HTTokens.color.backgroundTertiaryLight,
        hintText: hint,
        prefixIcon: Icon(
          Icons.search,
          color: isDark ? HTTokens.color.textTertiary : HTTokens.color.textTertiaryLight,
          size: HTTokens.size.iconMd,
        ),
        suffixIcon: controller != null && controller!.text.isNotEmpty
            ? IconButton(
                icon: Icon(
                  Icons.clear,
                  color: isDark ? HTTokens.color.textTertiary : HTTokens.color.textTertiaryLight,
                  size: HTTokens.size.iconMd,
                ),
                onPressed: onClear,
              )
            : null,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(HTTokens.border.radiusLg),
          borderSide: BorderSide.none,
        ),
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      ),
    );
  }
}

// ── Cards ────────────────────────────────────────────────

class HTCard extends StatelessWidget {
  final Widget child;
  final VoidCallback? onTap;
  final bool isSelected;
  final EdgeInsets padding;
  final List<BoxShadow>? shadows;

  const HTCard({
    super.key,
    required this.child,
    this.onTap,
    this.isSelected = false,
    this.padding = const EdgeInsets.all(16),
    this.shadows,
  });

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final t = HTTokens.color;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: padding,
        decoration: BoxDecoration(
          color: isDark ? t.backgroundElevated : t.backgroundElevatedLight,
          borderRadius: BorderRadius.circular(HTTokens.border.radiusLg),
          border: Border.all(
            color: isSelected
                ? t.brandPrimary
                : isDark
                    ? t.borderDefault
                    : t.borderDefaultLight,
            width: isSelected ? HTTokens.border.widthMedium : HTTokens.border.widthThin,
          ),
          boxShadow: shadows ??
              (isSelected ? HTTokens.shadow.glow : HTTokens.shadow.none),
        ),
        child: child,
      ),
    );
  }
}

// ── Status Indicators ────────────────────────────────────

class HTStatusDot extends StatelessWidget {
  final HTStatus status;
  final double size;
  final bool showPulse;

  const HTStatusDot({
    super.key,
    required this.status,
    this.size = 8,
    this.showPulse = true,
  });

  @override
  Widget build(BuildContext context) {
    final color = status.color;

    Widget dot = Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
      ),
    );

    if (showPulse && status == HTStatus.online) {
      dot = Stack(
        alignment: Alignment.center,
        children: [
          _PulsingDot(size: size * 2, color: color.withOpacity(0.3)),
          dot,
        ],
      );
    }

    return dot;
  }
}

class HTStatusChip extends StatelessWidget {
  final HTStatus status;
  final String label;

  const HTStatusChip({
    super.key,
    required this.status,
    required this.label,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: status.color.withOpacity(0.2),
        borderRadius: BorderRadius.circular(HTTokens.border.radiusFull),
      ),
      child: Text(
        label.toUpperCase(),
        style: TextStyle(
          fontSize: HTTokens.typography.sizeXs,
          fontWeight: HTTokens.typography.weightSemibold,
          color: status.color,
          letterSpacing: 0.5,
        ),
      ),
    );
  }
}

enum HTStatus {
  online,
  offline,
  error,
  warning,
  pending;

  Color get color {
    switch (this) {
      case HTStatus.online:
        return HTTokens.color.brandAccent;
      case HTStatus.offline:
        return HTTokens.color.textTertiary;
      case HTStatus.error:
        return HTTokens.color.brandDanger;
      case HTStatus.warning:
        return HTTokens.color.brandWarning;
      case HTStatus.pending:
        return HTTokens.color.brandInfo;
    }
  }
}

class _PulsingDot extends StatefulWidget {
  final double size;
  final Color color;

  const _PulsingDot({required this.size, required this.color});

  @override
  State<_PulsingDot> createState() => _PulsingDotState();
}

class _PulsingDotState extends State<_PulsingDot>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;
  late Animation<double> _animation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      duration: const Duration(seconds: 2),
      vsync: this,
    )..repeat(reverse: true);
    _animation = Tween<double>(begin: 0.3, end: 1.0).animate(_controller);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _animation,
      builder: (context, child) {
        return Container(
          width: widget.size,
          height: widget.size,
          decoration: BoxDecoration(
            color: widget.color.withOpacity(_animation.value),
            shape: BoxShape.circle,
          ),
        );
      },
    );
  }
}

// ── Empty State ──────────────────────────────────────────

class HTEmptyState extends StatelessWidget {
  final IconData icon;
  final String title;
  final String? subtitle;
  final Widget? action;

  const HTEmptyState({
    super.key,
    required this.icon,
    required this.title,
    this.subtitle,
    this.action,
  });

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              icon,
              size: HTTokens.size.iconXl,
              color: isDark ? HTTokens.color.textTertiary : HTTokens.color.textTertiaryLight,
            ),
            const SizedBox(height: 16),
            Text(
              title,
              style: Theme.of(context).textTheme.titleLarge,
              textAlign: TextAlign.center,
            ),
            if (subtitle != null) ...[
              const SizedBox(height: 8),
              Text(
                subtitle!,
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: isDark ? HTTokens.color.textSecondary : HTTokens.color.textSecondaryLight,
                    ),
                textAlign: TextAlign.center,
              ),
            ],
            if (action != null) ...[
              const SizedBox(height: 24),
              action!,
            ],
          ],
        ),
      ),
    );
  }
}

// ── Loading State ────────────────────────────────────────

class HTLoadingIndicator extends StatelessWidget {
  final String? message;

  const HTLoadingIndicator({super.key, this.message});

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          CircularProgressIndicator(
            color: HTTokens.color.brandPrimary,
            strokeWidth: 3,
          ),
          if (message != null) ...[
            const SizedBox(height: 16),
            Text(
              message!,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: isDark ? HTTokens.color.textSecondary : HTTokens.color.textSecondaryLight,
                  ),
            ),
          ],
        ],
      ),
    );
  }
}

// ── Divider with Label ───────────────────────────────────

class HTDividerWithLabel extends StatelessWidget {
  final String label;

  const HTDividerWithLabel({super.key, required this.label});

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    return Row(
      children: [
        Expanded(
          child: Divider(
            color: isDark ? HTTokens.color.borderDefault : HTTokens.color.borderDefaultLight,
          ),
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12),
          child: Text(
            label,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: isDark ? HTTokens.color.textTertiary : HTTokens.color.textTertiaryLight,
                ),
          ),
        ),
        Expanded(
          child: Divider(
            color: isDark ? HTTokens.color.borderDefault : HTTokens.color.borderDefaultLight,
          ),
        ),
      ],
    );
  }
}
