import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/billing_bloc.dart';
import '../widgets/empty_state.dart';
import '../widgets/loading_indicator.dart';
import '../widgets/error_widget.dart' as helix_error;
import '../widgets/plan_card.dart';
import '../widgets/metric_card.dart';

class BillingScreen extends StatefulWidget {
  const BillingScreen({super.key});

  @override
  State<BillingScreen> createState() => _BillingScreenState();
}

class _BillingScreenState extends State<BillingScreen> {
  @override
  void initState() {
    super.initState();
    context.read<BillingBloc>().add(BillingDashboardRequested());
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Billing'),
      ),
      body: BlocConsumer<BillingBloc, BillingState>(
        listener: (context, state) {
          if (state is BillingActionSuccess) {
            ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(state.message)));
          }
        },
        builder: (context, state) {
          if (state is BillingLoading) {
            return const LoadingIndicator();
          }
          if (state is BillingError) {
            return helix_error.ErrorWidget(
              message: state.message,
              onRetry: () => context.read<BillingBloc>().add(BillingDashboardRequested()),
            );
          }
          if (state is BillingDashboardLoaded) {
            return SingleChildScrollView(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Current Plan
                  Text('Current Plan', style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 12),
                  PlanCard(
                    name: state.plan['name'] ?? 'Free',
                    price: state.plan['price'] ?? '\$0',
                    features: List<String>.from(state.plan['features'] ?? []),
                    isCurrent: true,
                  ),
                  const SizedBox(height: 24),

                  // Usage Stats
                  Text('Usage Stats', style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 12),
                  GridView.count(
                    crossAxisCount: 2,
                    shrinkWrap: true,
                    physics: const NeverScrollableScrollPhysics(),
                    crossAxisSpacing: 12,
                    mainAxisSpacing: 12,
                    children: [
                      MetricCard(
                        title: 'Hosts',
                        value: '${state.usage['hosts'] ?? 0}',
                        icon: Icons.computer,
                      ),
                      MetricCard(
                        title: 'Sessions',
                        value: '${state.usage['sessions'] ?? 0}',
                        icon: Icons.terminal,
                      ),
                      MetricCard(
                        title: 'Storage',
                        value: '${state.usage['storage'] ?? 0} GB',
                        icon: Icons.storage,
                      ),
                      MetricCard(
                        title: 'Bandwidth',
                        value: '${state.usage['bandwidth'] ?? 0} GB',
                        icon: Icons.network_check,
                      ),
                    ],
                  ),
                  const SizedBox(height: 24),

                  // Payment Method
                  Text('Payment Method', style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 12),
                  Card(
                    child: ListTile(
                      leading: const Icon(Icons.credit_card),
                      title: Text(state.paymentMethod['brand'] ?? 'No payment method'),
                      subtitle: Text(state.paymentMethod['last4'] != null
                          ? '**** ${state.paymentMethod['last4']}'
                          : 'Add a payment method'),
                      trailing: TextButton(
                        onPressed: () {
                          // TODO: navigate to payment method update
                        },
                        child: const Text('Update'),
                      ),
                    ),
                  ),
                  const SizedBox(height: 24),

                  // Invoice List
                  Text('Invoices', style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 12),
                  if (state.invoices.isEmpty)
                    const EmptyState(message: 'No invoices yet'),
                  ...state.invoices.map((invoice) => Card(
                    child: ListTile(
                      leading: const Icon(Icons.receipt),
                      title: Text('Invoice #${invoice['number'] ?? 'N/A'}'),
                      subtitle: Text('${invoice['date'] ?? 'N/A'} - ${invoice['status'] ?? 'N/A'}'),
                      trailing: Text(
                        invoice['amount'] ?? '\$0',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      onTap: () {
                        // TODO: open invoice details
                      },
                    ),
                  )),
                ],
              ),
            );
          }
          return const SizedBox.shrink();
        },
      ),
    );
  }
}
